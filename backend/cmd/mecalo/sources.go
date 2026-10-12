package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/official"
	"github.com/chan-mai/mecalo/backend/internal/adapter/openfoodfacts"
	"github.com/chan-mai/mecalo/backend/internal/adapter/tabecal"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

// ビルド時に-ldflagsで上書き可
var version = "0.1.0"

// MECALO_SOURCESに指定できる取得元
var sourceNames = []string{
	"openfoodfacts", "openfoodfacts-delta", "tabecal",
	"topvalu", "familymart", "seven-eleven", "lawson",
	"mcdonalds", "starbucks", "mos", "ringerhut", "ootoya", "skylark", "saizeriya",
	"sushiro", "kurasushi", "godiva", "kourakuen", "matsunoya", "kiyoken", "cozycorner", "horai551",
	"bigboy", "doutor", "saintmarc",
}

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
		case "tabecal":
			deps.Finders = append(deps.Finders, tabecal.New("", cfg.TabecalKey, fetch.UserAgent))
		case "topvalu":
			deps.Catalogs = append(deps.Catalogs, &official.Topvalu{Fetch: fetch})
		case "familymart":
			deps.Catalogs = append(deps.Catalogs, &official.FamilyMart{Fetch: fetch})
		case "seven-eleven":
			deps.Catalogs = append(deps.Catalogs, &official.SevenEleven{Fetch: fetch})
		case "lawson":
			deps.Catalogs = append(deps.Catalogs, &official.Lawson{Fetch: fetch})
		case "mcdonalds":
			deps.Catalogs = append(deps.Catalogs, &official.McDonalds{Fetch: fetch})
		case "starbucks":
			deps.Catalogs = append(deps.Catalogs, &official.Starbucks{Fetch: fetch})
		case "mos":
			deps.Catalogs = append(deps.Catalogs, &official.Mos{Fetch: fetch})
		case "ringerhut":
			deps.Catalogs = append(deps.Catalogs, official.RingerHut(fetch))
		case "ootoya":
			deps.Catalogs = append(deps.Catalogs, official.Ootoya(fetch))
		case "skylark":
			deps.Catalogs = append(deps.Catalogs, &official.Skylark{Fetch: fetch})
		case "saizeriya":
			deps.Catalogs = append(deps.Catalogs, &official.Saizeriya{Fetch: fetch})
		case "sushiro":
			deps.Catalogs = append(deps.Catalogs, &official.Sushiro{Fetch: fetch})
		case "kurasushi":
			deps.Catalogs = append(deps.Catalogs, &official.KuraSushi{Fetch: fetch})
		case "godiva":
			deps.Catalogs = append(deps.Catalogs, &official.Godiva{Fetch: fetch})
		case "kourakuen":
			deps.Catalogs = append(deps.Catalogs, &official.Kourakuen{Fetch: fetch})
		case "matsunoya":
			deps.Catalogs = append(deps.Catalogs, &official.Matsunoya{Fetch: fetch})
		case "kiyoken":
			deps.Catalogs = append(deps.Catalogs, &official.Kiyoken{Fetch: fetch})
		case "cozycorner":
			deps.Catalogs = append(deps.Catalogs, &official.CozyCorner{Fetch: fetch})
		case "horai551":
			deps.Catalogs = append(deps.Catalogs, &official.Horai551{Fetch: fetch})
		case "bigboy":
			deps.Catalogs = append(deps.Catalogs, &official.BigBoy{Fetch: fetch})
		case "doutor":
			deps.Catalogs = append(deps.Catalogs, &official.Doutor{Fetch: fetch})
		case "saintmarc":
			deps.Catalogs = append(deps.Catalogs, &official.SaintMarc{Fetch: fetch})
		default:
			return fmt.Errorf("unknown source %q in MECALO_SOURCES (available: all, %s)", raw, strings.Join(sourceNames, ", "))
		}
	}
	return nil
}
