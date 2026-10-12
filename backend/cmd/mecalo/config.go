package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"time"
	_ "time/tzdata"

	"github.com/caarlos0/env/v11"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

// 起動時に環境変数からのみ読む設定
type bootConfig struct {
	Listen       string   `env:"MECALO_LISTEN" envDefault:":8080"`
	DatabaseURL  string   `env:"MECALO_DATABASE_URL" envDefault:"data/mecalo.db"`
	SecureCookie bool     `env:"MECALO_SECURE_COOKIE" envDefault:"true"`
	CORSOrigins  []string `env:"MECALO_CORS_ORIGINS"`
	// 空は起動時に生成
	SetupToken string `env:"MECALO_SETUP_TOKEN"`
}

func loadBootConfig() (bootConfig, error) {
	return env.ParseAs[bootConfig]()
}

// WebUIからも変更できる設定, 既定値はsettingDefs
type config struct {
	Location        *time.Location `env:"MECALO_TIMEZONE"`
	AuthProviders   []string       `env:"MECALO_AUTH_PROVIDERS"`
	RegistrationTo  string         `env:"MECALO_REGISTRATION"`
	OIDCName        string         `env:"MECALO_OIDC_NAME"`
	OIDCIssuer      string         `env:"MECALO_OIDC_ISSUER"`
	OIDCClientID    string         `env:"MECALO_OIDC_CLIENT_ID"`
	OIDCSecret      string         `env:"MECALO_OIDC_CLIENT_SECRET"`
	OIDCRedirectURL string         `env:"MECALO_OIDC_REDIRECT_URL"`
	OIDCScopes      []string       `env:"MECALO_OIDC_SCOPES"`
	OIDCAutoCreate  bool           `env:"MECALO_OIDC_AUTO_PROVISION"`
	OFFLookup       bool           `env:"MECALO_OPENFOODFACTS_LOOKUP"`
	OFFBaseURL      string         `env:"MECALO_OPENFOODFACTS_URL"`
	Sources         []string       `env:"MECALO_SOURCES"`
	TabecalKey      string         `env:"MECALO_TABECAL_KEY"`
	SyncInterval    time.Duration  `env:"MECALO_SYNC_INTERVAL"`
	FetchUserAgent  string         `env:"MECALO_FETCH_USER_AGENT"`
}

func (c config) LocalAuth() bool   { return slices.Contains(c.AuthProviders, "local") }
func (c config) OIDCEnabled() bool { return slices.Contains(c.AuthProviders, "oidc") }
func (c config) Registration() bool {
	return c.RegistrationTo == "open"
}

func settingDefs() []app.SettingDef {
	return []app.SettingDef{
		{Key: "MECALO_TIMEZONE", Group: "general", Kind: app.SettingText, Default: "Asia/Tokyo"},
		{Key: "MECALO_AUTH_PROVIDERS", Group: "auth", Kind: app.SettingChoices, Default: "local", Options: []string{"local", "oidc"}},
		{Key: "MECALO_REGISTRATION", Group: "auth", Kind: app.SettingChoice, Default: "closed", Options: []string{"closed", "open"}},
		{Key: "MECALO_OIDC_NAME", Group: "oidc", Kind: app.SettingText, Default: "OpenID Connect"},
		{Key: "MECALO_OIDC_ISSUER", Group: "oidc", Kind: app.SettingText},
		{Key: "MECALO_OIDC_CLIENT_ID", Group: "oidc", Kind: app.SettingText},
		{Key: "MECALO_OIDC_CLIENT_SECRET", Group: "oidc", Kind: app.SettingSecret},
		{Key: "MECALO_OIDC_REDIRECT_URL", Group: "oidc", Kind: app.SettingText},
		{Key: "MECALO_OIDC_SCOPES", Group: "oidc", Kind: app.SettingList, Default: "openid,profile"},
		{Key: "MECALO_OIDC_AUTO_PROVISION", Group: "oidc", Kind: app.SettingBool, Default: "false"},
		{Key: "MECALO_SOURCES", Group: "sources", Kind: app.SettingChoices, Options: append([]string{"all"}, sourceNames()...)},
		{Key: "MECALO_OPENFOODFACTS_LOOKUP", Group: "sources", Kind: app.SettingBool, Default: "false"},
		{Key: "MECALO_OPENFOODFACTS_URL", Group: "sources", Kind: app.SettingText},
		{Key: "MECALO_TABECAL_KEY", Group: "sources", Kind: app.SettingSecret},
		{Key: "MECALO_SYNC_INTERVAL", Group: "sources", Kind: app.SettingDuration, Default: "168h"},
		{Key: "MECALO_FETCH_USER_AGENT", Group: "sources", Kind: app.SettingText, Default: fmt.Sprintf(webfetch.DefaultUserAgent, version)},
	}
}

// 指定された環境変数のみ
func settingEnv(defs []app.SettingDef) map[string]string {
	out := map[string]string{}
	for _, d := range defs {
		if v := os.Getenv(d.Key); v != "" {
			out[d.Key] = v
		}
	}
	return out
}

// 画面・環境変数・既定値から決まった値の解釈
func parseConfig(values map[string]string) (config, error) {
	set := map[string]string{}
	for k, v := range values {
		if v != "" {
			set[k] = v
		}
	}
	c, err := env.ParseAsWithOptions[config](env.Options{Environment: set})
	if err != nil {
		return config{}, err
	}
	if c.Location == nil {
		return config{}, errors.New("MECALO_TIMEZONE is required")
	}
	if !c.LocalAuth() && !c.OIDCEnabled() {
		return config{}, errors.New("MECALO_AUTH_PROVIDERS must include local or oidc")
	}
	if c.OIDCEnabled() && (c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCRedirectURL == "") {
		return config{}, errors.New("MECALO_OIDC_ISSUER, MECALO_OIDC_CLIENT_ID and MECALO_OIDC_REDIRECT_URL are required for oidc")
	}
	return c, nil
}
