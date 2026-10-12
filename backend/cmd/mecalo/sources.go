package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/openfoodfacts"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

// ビルド時に-ldflagsで上書き可
var version = "0.1.0"

// MECALO_SOURCESに指定できる取得元
var sourceNames = []string{"openfoodfacts", "openfoodfacts-delta"}

func configureSources(cfg config, deps *app.Deps) error {
	names := slices.Clone(cfg.Sources)
	if cfg.OFFLookup {
		names = append(names, "openfoodfacts")
	}
	if slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(strings.TrimSpace(n), "all") }) {
		names = sourceNames
	}
	ua := cfg.FetchUserAgent
	if ua == "" {
		ua = fmt.Sprintf(webfetch.DefaultUserAgent, version)
	}
	fetch := webfetch.New(ua)
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		switch name {
		case "openfoodfacts":
			deps.Lookup = openfoodfacts.NewClient(cfg.OFFBaseURL, ua)
		case "openfoodfacts-delta":
			deps.Catalogs = append(deps.Catalogs, &openfoodfacts.Delta{Fetch: fetch})
		default:
			return fmt.Errorf("unknown source %q in MECALO_SOURCES (available: all, %s)", raw, strings.Join(sourceNames, ", "))
		}
	}
	return nil
}
