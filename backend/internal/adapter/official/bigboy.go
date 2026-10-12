package official

import (
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const bigboyBase = "https://www.bigboyjapan.co.jp"

// アレルゲン・カロリー画面が読むJSON, ソース単体も1件
type BigBoy struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*BigBoy)(nil)

const bigboyPath = "/allergen/menu/"

func (b *BigBoy) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "bigboy",
		Kind:        food.DatasetOfficial,
		Name:        "ビッグボーイ",
		License:     "bigboyjapan.co.jp (terms not found)",
		URL:         bigboyBase + bigboyPath,
		Attribution: "出典: ビッグボーイ アレルゲン・カロリー",
	}
}

type bigboyMenu struct {
	// 数値と「1928997n」等の文字列が混在
	ID    json.RawMessage `json:"メニューID"`
	Name  string          `json:"メニュー名"`
	Order float64         `json:"順番"`
	// ソース名ごとの同形式の値, 空は{}
	Sauces  json.RawMessage `json:"ソースリスト"`
	Energy  json.RawMessage `json:"熱量"`
	Protein json.RawMessage `json:"たんぱく質"`
	Fat     json.RawMessage `json:"脂質"`
	Carb    json.RawMessage `json:"炭水化物"`
	Salt    json.RawMessage `json:"食塩相当量"`
}

func (b *BigBoy) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(b.Base, "/")
	if base == "" {
		base = bigboyBase
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, b.Fetch, []string{base + bigboyPath + "json/menu.json"}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	foods, err := parseBigBoy(bodies[0])
	if err != nil {
		return prev, err
	}
	for _, sf := range foods {
		if err := sink.Emit(sf); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

// 掲載順のメニュー, 続けてソースのID順
func parseBigBoy(body []byte) ([]app.SystemFood, error) {
	var menus map[string]bigboyMenu
	if err := json.Unmarshal(body, &menus); err != nil {
		return nil, err
	}
	keys := slices.SortedFunc(maps.Keys(menus), func(a, b string) int {
		return cmp.Or(cmp.Compare(menus[a].Order, menus[b].Order), cmp.Compare(a, b))
	})
	var out []app.SystemFood
	sauces := map[string]app.SystemFood{}
	for _, key := range keys {
		m := menus[key]
		name := catalog.Normalize(m.Name)
		if name == "" {
			continue
		}
		if sf, ok := bigboyFood(key, name, m); ok {
			out = append(out, sf)
		}
		var list map[string]bigboyMenu
		_ = json.Unmarshal(m.Sauces, &list)
		for _, s := range list {
			// 末尾の「.」は同名ソースの区別用
			sauce := strings.TrimRight(catalog.Normalize(s.Name), ".")
			id := strings.Trim(string(s.ID), `"`)
			ref := "sauce-" + id
			if _, dup := sauces[ref]; dup || id == "" || id == "null" || sauce == "" {
				continue
			}
			// 最初に掲載されたメニューを用途として併記
			if sf, ok := bigboyFood(ref, sauce+"("+name+"用)", s); ok {
				sauces[ref] = sf
			}
		}
	}
	for _, ref := range slices.Sorted(maps.Keys(sauces)) {
		out = append(out, sauces[ref])
	}
	return out, nil
}

func bigboyFood(ref, name string, m bigboyMenu) (app.SystemFood, bool) {
	var v catalog.Values
	for _, f := range []struct {
		name string
		raw  json.RawMessage
		unit string
	}{
		{"熱量", m.Energy, "kcal"}, {"たんぱく質", m.Protein, "g"}, {"脂質", m.Fat, "g"},
		{"炭水化物", m.Carb, "g"}, {"食塩相当量", m.Salt, "g"},
	} {
		// 数値と文字列の両方を許容
		v.Set(f.name, strings.Trim(string(f.raw), `"`), f.unit)
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("bigboy", ref),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "ビッグボーイ",
		Reference: ref,
		Nutrients: v.Profile(),
		Basis:     food.Basis{Unit: "1食"},
		URL:       bigboyBase + bigboyPath,
	}, true
}
