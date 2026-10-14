package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi"
	"github.com/chan-mai/mecalo/backend/internal/adapter/oidc"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

// 設定の反映先
type liveState struct {
	app   *app.App
	clock *system.Clock
	// serve時のみOIDCを初期化
	serving  bool
	interval atomic.Int64
	auth     atomic.Pointer[httpapi.AuthMethods]
	// 前回と同じ設定ならプロバイダを再利用
	oidcConfig oidc.Config
	oidc       httpapi.OIDCProvider
}

func (l *liveState) syncInterval() time.Duration {
	return time.Duration(l.interval.Load())
}

func (l *liveState) authMethods() httpapi.AuthMethods {
	return *l.auth.Load()
}

// app.Instanceから直列に呼ばれる
func (l *liveState) prepare(ctx context.Context, values map[string]string) (func(), error) {
	cfg, err := parseConfig(values)
	if err != nil {
		return nil, err
	}
	if insecurePublicURL(cfg.PublicURL) {
		slog.Warn("MECALO_PUBLIC_URL uses http; OAuth tokens and codes are sent without encryption", "url", cfg.PublicURL)
	}
	var deps app.Deps
	if err := configureSources(cfg, &deps); err != nil {
		return nil, err
	}
	auth := httpapi.AuthMethods{Local: cfg.LocalAuth()}
	oc := oidc.Config{
		Name: cfg.OIDCName, Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID,
		ClientSecret: cfg.OIDCSecret, RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes,
	}
	if l.serving && cfg.OIDCEnabled() {
		if l.oidc != nil && sameOIDC(oc, l.oidcConfig) {
			auth.OIDC = l.oidc
		} else {
			p, err := oidc.New(ctx, oc)
			if err != nil {
				return nil, fmt.Errorf("MECALO_OIDC_ISSUER: %w", err)
			}
			auth.OIDC = p
		}
	}
	return func() {
		l.app.Reconfigure(app.Live{
			Options:  app.Options{RegistrationOpen: cfg.Registration(), OIDCAutoProvision: cfg.OIDCAutoCreate, MCPEnabled: cfg.MCPEnabled, PublicURL: cfg.PublicURL},
			Lookup:   deps.Lookup,
			Finders:  deps.Finders,
			Catalogs: deps.Catalogs,
		})
		l.clock.SetLocation(cfg.Location)
		l.interval.Store(int64(cfg.SyncInterval))
		l.auth.Store(&auth)
		l.oidc, l.oidcConfig = auth.OIDC, oc
	}, nil
}

func sameOIDC(a, b oidc.Config) bool {
	return a.Name == b.Name && a.Issuer == b.Issuer && a.ClientID == b.ClientID && a.ClientSecret == b.ClientSecret &&
		a.RedirectURL == b.RedirectURL && slices.Equal(a.Scopes, b.Scopes)
}
