package official

import (
	"context"
	"encoding/json"
	"html"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const starbucksBase = "https://product.starbucks.co.jp"

// 栄養成分一覧画面が読むJSON, 通常ミルクの全サイズ
type Starbucks struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Starbucks)(nil)

func (s *Starbucks) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "starbucks",
		Kind:        food.DatasetOfficial,
		Name:        "スターバックス",
		License:     "starbucks.co.jp terms of use (private use)",
		URL:         starbucksBase + "/allergy/nutrient/",
		Attribution: "出典: スターバックス コーヒー ジャパン 栄養成分一覧",
	}
}

type starbucksMenu struct {
	JAN       string                                  `json:"url_jan_code"`
	Link      string                                  `json:"link"`
	Name      string                                  `json:"product_name_ja"`
	Nutrition map[string][]map[string]json.RawMessage `json:"nutrition_by_milk"`
}

type starbucksKbn struct {
	Categories map[string]struct {
		Menus []starbucksMenu `json:"menu_groups"`
	} `json:"categories"`
}

// 店舗区分ごとの飲料・フード, 区分なしの配列は対象外
type starbucksBrand struct {
	Beverage json.RawMessage `json:"beverage"`
	Food     json.RawMessage `json:"food"`
}

var starbucksSizes = []string{"SHORT", "TALL", "GRANDE", "VENTI", "STANDARD"}

var starbucksFields = []struct {
	key  string
	code nutrient.Code
	unit string
}{
	{"calory", nutrient.Energy, "kcal"},
	{"protein", nutrient.Protein, "g"},
	{"lipid", nutrient.Fat, "g"},
	{"carbohydrate", nutrient.Carbohydrate, "g"},
	{"dietary_fiber", nutrient.Fiber, "g"},
	{"salt_eq", nutrient.SaltEquivalent, "g"},
	{"sodium", nutrient.Sodium, "mg"},
	{"potassium", nutrient.Potassium, "mg"},
}

func (s *Starbucks) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(s.Base, "/")
	if base == "" {
		base = starbucksBase
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, s.Fetch, []string{base + "/allergy/json/nutritions.json"}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	foods, err := parseStarbucks(bodies[0])
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

func parseStarbucks(body []byte) ([]app.SystemFood, error) {
	var brands map[string]starbucksBrand
	if err := json.Unmarshal(body, &brands); err != nil {
		return nil, err
	}
	byRef := map[string]app.SystemFood{}
	for _, brand := range []string{"STARBUCKS", "RESERVE"} {
		b := brands[brand]
		for kind, raw := range map[string]json.RawMessage{"beverage": b.Beverage, "food": b.Food} {
			var kbns map[string]starbucksKbn
			if json.Unmarshal(raw, &kbns) != nil {
				continue
			}
			for _, kbn := range kbns {
				for _, cat := range kbn.Categories {
					for _, m := range cat.Menus {
						for _, sf := range starbucksFoods(m, kind) {
							if _, dup := byRef[sf.Reference]; !dup {
								byRef[sf.Reference] = sf
							}
						}
					}
				}
			}
		}
	}
	refs := make([]string, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	out := make([]app.SystemFood, 0, len(refs))
	for _, ref := range refs {
		out = append(out, byRef[ref])
	}
	return out, nil
}

func starbucksFoods(m starbucksMenu, kind string) []app.SystemFood {
	name := catalog.Normalize(html.UnescapeString(m.Name))
	if m.JAN == "" || name == "" {
		return nil
	}
	unit := "1杯"
	if kind == "food" {
		unit = "1個"
	}
	var out []app.SystemFood
	for _, size := range starbucksSizes {
		rows := m.Nutrition[size]
		var row map[string]json.RawMessage
		for _, r := range rows {
			// 通常ミルクのみ, ミルク選択なしはnull
			var milk *string
			_ = json.Unmarshal(r["_milk_type"], &milk)
			if milk == nil || *milk == "milk" {
				row = r
				break
			}
		}
		if row == nil {
			continue
		}
		var v catalog.Values
		for _, f := range starbucksFields {
			var x float64
			if json.Unmarshal(row[f.key], &x) == nil && x >= 0 {
				v.SetNumber(f.code, decimal.NewFromFloat(x).Round(2), f.unit)
			}
		}
		if v.Len() == 0 {
			continue
		}
		label, ref := name, m.JAN
		if size != "STANDARD" {
			label = name + " " + size[:1] + strings.ToLower(size[1:])
			ref = m.JAN + "-" + strings.ToLower(size)
		}
		out = append(out, app.SystemFood{
			ID:        food.SystemID("starbucks", ref),
			Kind:      food.KindProduct,
			Name:      label,
			Brand:     "スターバックス",
			Reference: ref,
			Nutrients: v.Profile(),
			Basis:     food.Basis{Unit: unit},
			URL:       "https://menu.starbucks.co.jp/" + m.JAN,
		})
	}
	return out
}
