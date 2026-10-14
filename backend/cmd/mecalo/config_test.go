package main

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// 設定項目と構造体の環境変数名の一致
func TestSettingDefs(t *testing.T) {
	var keys []string
	for _, d := range settingDefs() {
		keys = append(keys, d.Key)
	}
	typ := reflect.TypeFor[config]()
	var tags []string
	for i := range typ.NumField() {
		tags = append(tags, typ.Field(i).Tag.Get("env"))
	}
	slices.Sort(keys)
	slices.Sort(tags)
	if !slices.Equal(keys, tags) {
		t.Errorf("defs = %v, fields = %v", keys, tags)
	}

	values := map[string]string{}
	for _, d := range settingDefs() {
		values[d.Key] = d.Default
	}
	c, err := parseConfig(values)
	if err != nil {
		t.Fatal(err)
	}
	if c.Location.String() != "Asia/Tokyo" || c.SyncInterval != 168*time.Hour || !c.LocalAuth() || c.OIDCEnabled() || c.Registration() {
		t.Errorf("defaults = %+v", c)
	}

	t.Setenv("MECALO_TIMEZONE", "")
	t.Setenv("MECALO_SYNC_INTERVAL", "24h")
	if env := settingEnv(settingDefs()); len(env) != 1 || env["MECALO_SYNC_INTERVAL"] != "24h" {
		t.Errorf("env = %v", env)
	}

	values["MECALO_AUTH_PROVIDERS"] = "oidc"
	if _, err := parseConfig(values); err == nil {
		t.Error("oidc without issuer accepted")
	}
	values["MECALO_AUTH_PROVIDERS"] = "local"
	values["MECALO_TIMEZONE"] = "Mars/Base"
	if _, err := parseConfig(values); err == nil {
		t.Error("unknown timezone accepted")
	}
}

func TestPublicURL(t *testing.T) {
	ok := map[string]string{
		"":                            "",
		"https://Mecalo.Example.com/": "https://mecalo.example.com",
		"http://localhost:3000":       "http://localhost:3000",
	}
	for in, want := range ok {
		if got, err := publicURL(in); err != nil || got != want {
			t.Errorf("publicURL(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"mecalo.example.com", "ftp://example.com", "https://example.com/mecalo", "https://example.com?x=1"} {
		if _, err := publicURL(in); err == nil {
			t.Errorf("publicURL(%q) accepted", in)
		}
	}
}
