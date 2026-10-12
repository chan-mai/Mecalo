package official

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const mosBase = "https://www.mos.jp"

// 栄養成分画面が読むカテゴリ別JSON
type Mos struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Mos)(nil)

func (m *Mos) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "mos",
		Kind:        food.DatasetOfficial,
		Name:        "モスバーガー",
		License:     "mos.jp terms of use (private use)",
		URL:         mosBase + "/menu/",
		Attribution: "出典: モスバーガー 栄養成分",
	}
}

var mosCategory = regexp.MustCompile(`menu_category/(\d+)\.json`)

// サイズ番号と表示名
var mosSizes = map[string]string{"1": "S", "2": "M", "3": "L", "4": "プチ"}

type mosNutrition struct {
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
}

type mosMenu struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Nutrition []mosNutrition `json:"nutrition"`
	// サイズ別の値, 形式不明のため緩く解釈
	Group json.RawMessage `json:"group"`
}

func (m *Mos) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(m.Base, "/")
	if base == "" {
		base = mosBase
	}
	script, err := m.Fetch.Get(ctx, base+"/common/menu/js/nutrition.js", webfetch.Validators{})
	if err != nil {
		return prev, err
	}
	var urls []string
	seen := map[string]bool{}
	for _, c := range mosCategory.FindAllStringSubmatch(string(script.Body), -1) {
		if !seen[c[1]] {
			seen[c[1]] = true
			urls = append(urls, base+"/data/menu/menu_category/"+c[1]+".json")
		}
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, m.Fetch, urls, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	emitted := map[food.ID]bool{}
	for _, body := range bodies {
		foods, err := parseMos(body)
		if err != nil {
			return prev, err
		}
		for _, sf := range foods {
			if emitted[sf.ID] {
				continue
			}
			emitted[sf.ID] = true
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

func parseMos(body []byte) ([]app.SystemFood, error) {
	var menus []mosMenu
	if err := json.Unmarshal(body, &menus); err != nil {
		return nil, err
	}
	var out []app.SystemFood
	for _, m := range menus {
		name := extract.Normalize(m.Name)
		if m.ID == "" || name == "" {
			continue
		}
		if sf, ok := mosFood(m.ID, name, m.Nutrition); ok {
			out = append(out, sf)
		}
		var groups map[string]json.RawMessage
		_ = json.Unmarshal(m.Group, &groups)
		keys := make([]string, 0, len(groups))
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var g struct {
				Nutrition []mosNutrition `json:"nutrition"`
			}
			size, ok := mosSizes[k]
			if !ok || json.Unmarshal(groups[k], &g) != nil {
				continue
			}
			if sf, ok := mosFood(m.ID+"-"+k, name+" "+size, g.Nutrition); ok {
				out = append(out, sf)
			}
		}
	}
	return out, nil
}

func mosFood(ref, name string, rows []mosNutrition) (app.SystemFood, bool) {
	var v extract.Values
	basis := food.Basis{Unit: "1食"}
	for _, n := range rows {
		amount, unit := extract.SplitValue(n.Quantity)
		if extract.Normalize(n.Name) == "重量" {
			if g, err := decimal.NewFromString(amount); err == nil && g.IsPositive() {
				basis.Grams = g
			}
			continue
		}
		v.Set(n.Name, amount, unit)
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("mos", ref),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "モスバーガー",
		Reference: ref,
		Nutrients: v.Profile(),
		Basis:     basis,
		URL:       mosBase + "/menu/",
	}, true
}
