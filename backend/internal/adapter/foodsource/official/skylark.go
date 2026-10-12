package official

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const skylarkBase = "https://www.skylark.co.jp"

// すかいらーく各ブランドのメニューJSON, 公表値はエネルギーと食塩のみ
type Skylark struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Skylark)(nil)

// URLの識別子と表示名
var skylarkBrands = []struct{ slug, name string }{
	{"gusto", "ガスト"}, {"bamiyan", "バーミヤン"}, {"jonathan", "ジョナサン"},
	{"yumean", "夢庵"}, {"karayoshi", "から好し"},
}

func (s *Skylark) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "skylark",
		Kind:        food.DatasetOfficial,
		Name:        "すかいらーく",
		License:     "skylark.co.jp site policy (private use)",
		URL:         skylarkBase + "/",
		Attribution: "出典: すかいらーくグループ 各ブランドのメニュー情報",
	}
}

type skylarkMenu struct {
	UID        string `json:"menu_uid"`
	Name       string `json:"menu_name"`
	Variations []struct {
		Name    string `json:"name"`
		Calorie string `json:"calorie"`
		Salt    string `json:"salt"`
	} `json:"valiation"`
}

func (s *Skylark) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(s.Base, "/")
	if base == "" {
		base = skylarkBase
	}
	urls := make([]string, 0, len(skylarkBrands))
	for _, b := range skylarkBrands {
		urls = append(urls, base+"/"+b.slug+"/menu/json/menu_detail.json")
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, s.Fetch, urls, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	for i, body := range bodies {
		foods, err := parseSkylark(body, skylarkBrands[i].slug, skylarkBrands[i].name)
		if err != nil {
			return prev, err
		}
		for _, sf := range foods {
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

func parseSkylark(body []byte, slug, brand string) ([]app.SystemFood, error) {
	var doc struct {
		List []skylarkMenu `json:"list"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	var out []app.SystemFood
	seen := map[string]bool{}
	for _, m := range doc.List {
		name := extract.Normalize(m.Name)
		if m.UID == "" || name == "" {
			continue
		}
		for i, v := range m.Variations {
			var values extract.Values
			values.Set("エネルギー", v.Calorie, "kcal")
			values.Set("食塩相当量", v.Salt, "g")
			if values.Len() == 0 {
				continue
			}
			label, ref := name, slug+"-"+m.UID
			if len(m.Variations) > 1 {
				ref = ref + "-" + string(rune('a'+i))
				if n := extract.Normalize(v.Name); n != "" {
					label = name + " " + n
				}
			}
			// 同じ商品が複数カテゴリに掲載
			if seen[ref] {
				continue
			}
			seen[ref] = true
			out = append(out, app.SystemFood{
				ID:        food.SystemID("skylark", ref),
				Kind:      food.KindProduct,
				Name:      label,
				Brand:     brand,
				Reference: ref,
				Nutrients: values.Profile(),
				Basis:     food.Basis{Unit: "1食"},
				URL:       skylarkBase + "/" + slug + "/menu/",
			})
		}
	}
	return out, nil
}

const saizeriyaURL = "https://allergy.saizeriya.co.jp/view?lg=ja"

// アレルギー情報画面が読むJSON, 公表値はエネルギーと食塩のみ
type Saizeriya struct {
	Fetch *webfetch.Client
	// テスト用の接続先
	URL string
}

var _ port.Catalog = (*Saizeriya)(nil)

func (s *Saizeriya) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "saizeriya",
		Kind:        food.DatasetOfficial,
		Name:        "サイゼリヤ",
		License:     "saizeriya.co.jp terms of use",
		URL:         saizeriyaURL,
		Attribution: "出典: サイゼリヤ アレルギー・栄養情報",
	}
}

func (s *Saizeriya) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	u := s.URL
	if u == "" {
		u = saizeriyaURL
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, s.Fetch, []string{u}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	foods, err := parseSaizeriya(bodies[0])
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

func parseSaizeriya(body []byte) ([]app.SystemFood, error) {
	var doc struct {
		Body []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Calorie string `json:"calorie"`
			Salt    string `json:"salt"`
		} `json:"body"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	var out []app.SystemFood
	for _, m := range doc.Body {
		name := extract.Normalize(m.Name)
		var values extract.Values
		values.Set("エネルギー", m.Calorie, "kcal")
		values.Set("食塩相当量", m.Salt, "g")
		if m.ID == "" || name == "" || values.Len() == 0 {
			continue
		}
		out = append(out, app.SystemFood{
			ID:        food.SystemID("saizeriya", m.ID),
			Kind:      food.KindProduct,
			Name:      name,
			Brand:     "サイゼリヤ",
			Reference: m.ID,
			Nutrients: values.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       saizeriyaURL,
		})
	}
	return out, nil
}
