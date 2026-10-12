package main

import (
	"errors"
	"slices"
	"time"
	_ "time/tzdata"

	"github.com/caarlos0/env/v11"
)

// 環境変数からの設定
type config struct {
	Listen          string         `env:"MECALO_LISTEN" envDefault:":8080"`
	DatabaseURL     string         `env:"MECALO_DATABASE_URL" envDefault:"data/mecalo.db"`
	Location        *time.Location `env:"MECALO_TIMEZONE" envDefault:"Asia/Tokyo"`
	AuthProviders   []string       `env:"MECALO_AUTH_PROVIDERS" envDefault:"local"`
	RegistrationTo  string         `env:"MECALO_REGISTRATION" envDefault:"closed"`
	SecureCookie    bool           `env:"MECALO_SECURE_COOKIE" envDefault:"true"`
	CORSOrigins     []string       `env:"MECALO_CORS_ORIGINS"`
	OIDCName        string         `env:"MECALO_OIDC_NAME" envDefault:"OpenID Connect"`
	OIDCIssuer      string         `env:"MECALO_OIDC_ISSUER"`
	OIDCClientID    string         `env:"MECALO_OIDC_CLIENT_ID"`
	OIDCSecret      string         `env:"MECALO_OIDC_CLIENT_SECRET"`
	OIDCRedirectURL string         `env:"MECALO_OIDC_REDIRECT_URL"`
	OIDCScopes      []string       `env:"MECALO_OIDC_SCOPES" envDefault:"openid,profile"`
	OIDCAutoCreate  bool           `env:"MECALO_OIDC_AUTO_PROVISION"`
	OFFLookup       bool           `env:"MECALO_OPENFOODFACTS_LOOKUP"`
	OFFBaseURL      string         `env:"MECALO_OPENFOODFACTS_URL"`
	Sources         []string       `env:"MECALO_SOURCES"`
	TabecalKey      string         `env:"MECALO_TABECAL_KEY"`
	SyncInterval    time.Duration  `env:"MECALO_SYNC_INTERVAL" envDefault:"168h"`
	FetchUserAgent  string         `env:"MECALO_FETCH_USER_AGENT"`
}

func (c config) LocalAuth() bool   { return slices.Contains(c.AuthProviders, "local") }
func (c config) OIDCEnabled() bool { return slices.Contains(c.AuthProviders, "oidc") }
func (c config) Registration() bool {
	return c.RegistrationTo == "open"
}

func loadConfig() (config, error) {
	c, err := env.ParseAs[config]()
	if err != nil {
		return config{}, err
	}
	if !c.LocalAuth() && !c.OIDCEnabled() {
		return config{}, errors.New("MECALO_AUTH_PROVIDERS must include local or oidc")
	}
	if c.OIDCEnabled() && (c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCRedirectURL == "") {
		return config{}, errors.New("MECALO_OIDC_ISSUER, MECALO_OIDC_CLIENT_ID and MECALO_OIDC_REDIRECT_URL are required for oidc")
	}
	return c, nil
}
