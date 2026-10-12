// SQLite / CockroachDB共通のレポジトリ実装
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/chan-mai/mecalo/backend/internal/port"
	"github.com/chan-mai/mecalo/backend/migrations"
)

const maxTxAttempts = 3

type Store struct {
	db  *sql.DB
	d   dialect
	idx indexer
	repos
}

var _ port.Store = (*Store)(nil)

// pathはファイルパス, ":memory:" は不可
type Option func(*Store)

// 食品名の読みでの検索を有効化
func WithReader(r port.TextReader) Option {
	return func(s *Store) { s.idx.reader = r }
}

func OpenSQLite(ctx context.Context, path string, opts ...Option) (*Store, error) {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Set("_txlock", "immediate")
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return open(ctx, db, sqliteDialect, goose.DialectSQLite3, migrations.SQLite, "sqlite", opts)
}

func OpenPostgres(ctx context.Context, dsn string, opts ...Option) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return open(ctx, db, postgresDialect, goose.DialectPostgres, migrations.Postgres, "postgres", opts)
}

func open(ctx context.Context, db *sql.DB, d dialect, gd goose.Dialect, fsys fs.FS, dir string, opts []Option) (*Store, error) {
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		db.Close()
		return nil, err
	}
	provider, err := goose.NewProvider(gd, db, sub)
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err := provider.Up(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	s := &Store{db: db, d: d}
	for _, o := range opts {
		o(s)
	}
	s.repos = repos{q: conn{db: db, d: d}, d: d, idx: s.idx}
	if err := s.reindex(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("reindex foods: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InTx(ctx context.Context, fn func(port.Repositories) error) error {
	var err error
	for range maxTxAttempts {
		err = s.inTxOnce(ctx, fn)
		if err == nil || !s.d.retryable(err) {
			return err
		}
	}
	return err
}

func (s *Store) inTxOnce(ctx context.Context, fn func(port.Repositories) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(repos{q: conn{db: tx, d: s.d}, d: s.d, idx: s.idx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// SQLiteのオンラインバックアップ
func (s *Store) BackupTo(ctx context.Context, path string) error {
	if s.d.name != sqliteDialect.name {
		return errors.New("backup is only supported for sqlite; use the database's own tools")
	}
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", path)
	return err
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// プレースホルダを方言へ変換して実行
type conn struct {
	db execer
	d  dialect
}

func (c conn) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.db.ExecContext(ctx, c.d.rebind(query), args...)
}

func (c conn) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, c.d.rebind(query), args...)
}

// 全行を走査しrowsは必ず閉じる
func (c conn) each(ctx context.Context, query string, args []any, fn func(*sql.Rows) error) error {
	rows, err := c.query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (c conn) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return c.db.QueryRowContext(ctx, c.d.rebind(query), args...)
}

type repos struct {
	q   conn
	d   dialect
	idx indexer
}

func (r repos) Users() port.UserRepository                  { return users(r) }
func (r repos) Sessions() port.SessionRepository            { return sessions(r) }
func (r repos) Datasets() port.DatasetRepository            { return datasets(r) }
func (r repos) Foods() port.FoodRepository                  { return foods(r) }
func (r repos) Recipes() port.RecipeRepository              { return recipes(r) }
func (r repos) Meals() port.MealRepository                  { return meals(r) }
func (r repos) Favorites() port.FavoriteRepository          { return favorites(r) }
func (r repos) SearchHistory() port.SearchHistoryRepository { return searchHistory(r) }
func (r repos) QueryLinks() port.QueryLinkRepository        { return queryLinks(r) }
func (r repos) Settings() port.SettingsRepository           { return settings(r) }
func (r repos) Weights() port.WeightRepository              { return weights(r) }
func (r repos) Sources() port.SourceRepository              { return sources(r) }

// ゼロ値はNULL
func nullTimeArg(d dialect, t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return d.timeValue(t)
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return port.ErrNotFound
	}
	return err
}

func affectedOne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}
