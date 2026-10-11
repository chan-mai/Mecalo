// Mecaloサーバとメンテナンス用コマンド
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chan-mai/mecalo/backend/data"
	"github.com/chan-mai/mecalo/backend/internal/adapter/dri"
	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi"
	"github.com/chan-mai/mecalo/backend/internal/adapter/mext"
	"github.com/chan-mai/mecalo/backend/internal/adapter/oidc"
	"github.com/chan-mai/mecalo/backend/internal/adapter/openfoodfacts"
	"github.com/chan-mai/mecalo/backend/internal/adapter/password"
	"github.com/chan-mai/mecalo/backend/internal/adapter/reading"
	"github.com/chan-mai/mecalo/backend/internal/adapter/sqlstore"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

const usage = `Usage: mecalo <command> [options]

Commands:
  serve                         Start the API server
  import mext [-file path]      Import the Standard Tables of Food Composition in Japan
  import off -file dump.jsonl   Import Open Food Facts products from a JSONL dump
  user create -login name       Create a local user (-admin for administrator)
  backup -out path              Write an online SQLite backup
  export -login name            Write a user's data as JSON to stdout

Configuration is read from MECALO_* environment variables. See README.md.
`

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	if err := run(os.Args[1:]); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return errors.New("command is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch args[0] {
	case "serve":
		return serve(ctx, cfg)
	case "import":
		if len(args) < 2 {
			return errors.New("import target is required: mext or off")
		}
		switch args[1] {
		case "mext":
			return importMEXT(ctx, cfg, args[2:])
		case "off":
			return importOFF(ctx, cfg, args[2:])
		}
		return fmt.Errorf("unknown import target %q", args[1])
	case "user":
		if len(args) < 2 || args[1] != "create" {
			return errors.New("usage: mecalo user create -login name")
		}
		return createUser(ctx, cfg, args[2:])
	case "backup":
		return backup(ctx, cfg, args[1:])
	case "export":
		return export(ctx, cfg, args[1:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	}
	return fmt.Errorf("unknown command %q", args[0])
}

type runtime struct {
	app   *app.App
	store *sqlstore.Store
}

func open(ctx context.Context, cfg config) (*runtime, error) {
	reader := reading.New()
	store, err := openStore(ctx, cfg.DatabaseURL, sqlstore.WithReader(reader))
	if err != nil {
		return nil, err
	}
	ref, err := dri.Load(data.DRI2025)
	if err != nil {
		store.Close()
		return nil, err
	}
	deps := app.Deps{
		Store:     store,
		Reader:    reader,
		Clock:     system.Clock{Location: cfg.Location},
		IDs:       system.UUIDs{},
		Reference: ref,
		Hasher:    password.Default(),
	}
	if cfg.OFFLookup {
		deps.Barcode = openfoodfacts.NewClient(cfg.OFFBaseURL)
	}
	a := app.New(deps, app.Options{RegistrationOpen: cfg.Registration(), OIDCAutoProvision: cfg.OIDCAutoCreate})
	return &runtime{app: a, store: store}, nil
}

func openStore(ctx context.Context, url string, opts ...sqlstore.Option) (*sqlstore.Store, error) {
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return sqlstore.OpenPostgres(ctx, url, opts...)
	}
	path := strings.TrimPrefix(url, "sqlite:")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	return sqlstore.OpenSQLite(ctx, path, opts...)
}

func serve(ctx context.Context, cfg config) error {
	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()

	opt := httpapi.Options{
		LocalAuth:      cfg.LocalAuth(),
		SecureCookie:   cfg.SecureCookie,
		AllowedOrigins: cfg.CORSOrigins,
		Logger:         slog.Default(),
	}
	if cfg.OIDCEnabled() {
		p, err := oidc.New(ctx, oidc.Config{
			Name: cfg.OIDCName, Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID,
			ClientSecret: cfg.OIDCSecret, RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes,
		})
		if err != nil {
			return err
		}
		opt.OIDC = p
	}
	handler, err := httpapi.NewHandler(rt.app, opt)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go purgeSessions(ctx, rt.app)

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Listen)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

func purgeSessions(ctx context.Context, a *app.App) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := a.Accounts.PurgeExpiredSessions(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("purge sessions", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func importMEXT(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("import mext", flag.ContinueOnError)
	file := fs.String("file", "", "local xlsx path (downloads from MEXT when empty)")
	url := fs.String("url", mext.DefaultURL, "download URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var r io.ReadCloser
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		r = f
	} else {
		slog.Info("downloading", "url", *url)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, *url, nil)
		if err != nil {
			return err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return fmt.Errorf("download failed: status %d", res.StatusCode)
		}
		r = res.Body
	}
	table, err := mext.Parse(r)
	r.Close()
	if err != nil {
		return err
	}

	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()
	stats, err := rt.app.Imports.Upsert(ctx, mext.Dataset(table.UpdatedAt), table.Foods)
	if err != nil {
		return err
	}
	slog.Info("imported mext", "version", table.UpdatedAt, "created", stats.Created, "updated", stats.Updated, "unchanged", stats.Unchanged)
	return nil
}

func importOFF(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("import off", flag.ContinueOnError)
	file := fs.String("file", "", "Open Food Facts JSONL dump (.jsonl or .jsonl.gz)")
	country := fs.String("country", "japan", "countries_tags value without the en: prefix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("-file is required")
	}
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer f.Close()

	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()

	var batch []app.SystemFood
	var total app.ImportStats
	flush := func() error {
		stats, err := rt.app.Imports.Upsert(ctx, openfoodfacts.Dataset(), batch)
		total.Created += stats.Created
		total.Updated += stats.Updated
		total.Unchanged += stats.Unchanged
		batch = batch[:0]
		return err
	}
	err = openfoodfacts.ReadDump(f, *country, func(sf app.SystemFood) error {
		batch = append(batch, sf)
		if len(batch) >= 5000 {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return err
	}
	slog.Info("imported openfoodfacts", "created", total.Created, "updated", total.Updated, "unchanged", total.Unchanged)
	return nil
}

func createUser(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	login := fs.String("login", "", "login name")
	name := fs.String("name", "", "display name")
	admin := fs.Bool("admin", false, "grant administrator role")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pw := os.Getenv("MECALO_NEW_PASSWORD")
	if pw == "" {
		return errors.New("set the password in MECALO_NEW_PASSWORD")
	}
	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()
	role := user.RoleMember
	if *admin {
		role = user.RoleAdmin
	}
	u, err := rt.app.Accounts.CreateUserUnchecked(ctx, app.SignUpInput{LoginName: *login, DisplayName: *name, Password: pw}, role)
	if err != nil {
		return err
	}
	slog.Info("user created", "id", u.ID, "role", u.Role)
	return nil
}

func backup(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	out := fs.String("out", "", "destination file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	if _, err := os.Stat(*out); err == nil {
		return fmt.Errorf("%s already exists", *out)
	}
	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()
	if err := rt.store.BackupTo(ctx, *out); err != nil {
		return err
	}
	slog.Info("backup written", "path", *out)
	return nil
}

func export(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	login := fs.String("login", "", "local login name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rt, err := open(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.store.Close()
	u, _, err := rt.store.Users().FindByCredential(ctx, app.ProviderLocal, strings.ToLower(*login))
	if err != nil {
		return fmt.Errorf("user %q: %w", *login, err)
	}
	b, err := rt.app.Exports.Export(ctx, u)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(httpapi.ToBundle(b))
}
