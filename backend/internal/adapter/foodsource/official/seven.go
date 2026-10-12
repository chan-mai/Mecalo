package official

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const sevenBase = "https://www.sej.co.jp"

// 商品カテゴリの一覧と地域別の商品詳細
type SevenEleven struct {
	Fetch    *webfetch.Client
	Base     string
	MaxPages int
}

var _ port.Catalog = (*SevenEleven)(nil)

func (s *SevenEleven) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "seven-eleven",
		Kind:        food.DatasetOfficial,
		Name:        "セブン‐イレブン",
		License:     "sej.co.jp terms of use (private use)",
		URL:         sevenBase + "/products/",
		Attribution: "出典: セブン‐イレブン 商品情報(sej.co.jp)",
	}
}

var sevenCategories = []string{
	"onigiri", "sushi", "bento", "sandwich", "bread", "pasta", "men", "gratin", "dailydish", "salad",
	"sweets", "ice_cream", "frozen_foods", "hotsnack", "chukaman", "oden", "donut", "sevencafe", "7premium",
}

var sevenItem = regexp.MustCompile(`^/products/a/item/(\d{6})/(?:([a-z]+)/)?$`)

func (s *SevenEleven) base() string {
	if s.Base != "" {
		return strings.TrimRight(s.Base, "/")
	}
	return sevenBase
}

func (s *SevenEleven) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	roots := make([]string, 0, len(sevenCategories))
	for _, c := range sevenCategories {
		roots = append(roots, "/products/a/"+c+"/")
	}
	c := crawler{
		fetch: s.Fetch,
		base:  s.base(),
		follow: func(p string) bool {
			return strings.HasPrefix(p, "/products/a/") && !strings.HasPrefix(p, "/products/a/item/") &&
				!strings.Contains(p, "thisweek") && !strings.Contains(p, "nextweek")
		},
		item:     sevenItem,
		maxPages: s.maxPages(),
		hosts:    []string{"www.sej.co.jp"},
	}
	paths, err := c.collect(ctx, roots)
	if err != nil {
		return prev, err
	}
	byCode := regionPages(paths)
	for _, code := range slices.Sorted(maps.Keys(byCode)) {
		ids := sevenIDs(code)
		if skipped, err := seenFresh(sink, ids); skipped || err != nil {
			if err != nil {
				return prev, err
			}
			continue
		}
		var (
			pages  []sevenPage
			failed error
		)
		for _, page := range byCode[code] {
			res, err := s.Fetch.Get(ctx, s.base()+page.path, webfetch.Validators{})
			if errors.Is(err, webfetch.ErrNotFound) {
				continue
			}
			if err != nil {
				if ctx.Err() != nil {
					return prev, err
				}
				failed = err
				continue
			}
			doc, err := extract.ParseHTML(res.Body)
			if err != nil {
				return prev, err
			}
			if sf, ok := parseSeven(doc, code); ok {
				page.food = sf
				pages = append(pages, page)
			}
		}
		// 一部の地域が取得できない場合は前回の内容を維持
		if failed != nil {
			if err := failAll(sink, ids, failed); err != nil {
				return prev, err
			}
			continue
		}
		for _, sf := range groupRegions(code, pages) {
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{}, nil
}

func (s *SevenEleven) maxPages() int {
	if s.MaxPages > 0 {
		return s.MaxPages
	}
	return 2000
}

// 販売地域の識別子と表示名, 表示順
var sevenAreas = []struct{ slug, name string }{
	{"hokkaido", "北海道"}, {"tohoku", "東北"}, {"kanto", "関東"}, {"koshinetsu", "甲信越"}, {"hokuriku", "北陸"},
	{"tokai", "東海"}, {"kinki", "近畿"}, {"chugoku", "中国"}, {"shikoku", "四国"}, {"kyushu", "九州"}, {"okinawa", "沖縄"},
}

type sevenPage struct {
	path string
	// 空は地域指定なし
	area string
	food app.SystemFood
}

// 商品コードごとの地域別ページ, 地域の表示順
func regionPages(paths []string) map[string][]sevenPage {
	order := map[string]int{"": -1}
	for i, a := range sevenAreas {
		order[a.slug] = i
	}
	out := map[string][]sevenPage{}
	for _, p := range paths {
		m := sevenItem.FindStringSubmatch(p)
		if m == nil {
			continue
		}
		if _, known := order[m[2]]; !known {
			continue
		}
		out[m[1]] = append(out[m[1]], sevenPage{path: p, area: m[2]})
	}
	for code := range out {
		slices.SortFunc(out[code], func(a, b sevenPage) int { return order[a.area] - order[b.area] })
	}
	return out
}

// 地域指定なしと全地域の候補ID
func sevenIDs(code string) []food.ID {
	ids := []food.ID{food.SystemID("seven-eleven", code)}
	for _, a := range sevenAreas {
		ids = append(ids, food.SystemID("seven-eleven", code+"-"+a.slug))
	}
	return ids
}

// 全地域で同じ値は1件, 異なる値は地域を併記して値ごとに1件
func groupRegions(code string, pages []sevenPage) []app.SystemFood {
	type group struct {
		food  app.SystemFood
		areas []string
		path  string
	}
	var groups []*group
	for _, p := range pages {
		var g *group
		for _, x := range groups {
			if sameValues(x.food, p.food) {
				g = x
				break
			}
		}
		if g == nil {
			g = &group{food: p.food, path: p.path}
			groups = append(groups, g)
		}
		g.areas = append(g.areas, p.area)
	}
	out := make([]app.SystemFood, 0, len(groups))
	for _, g := range groups {
		sf := g.food
		sf.URL = sevenBase + g.path
		if len(groups) > 1 {
			names := make([]string, 0, len(g.areas))
			for _, a := range g.areas {
				names = append(names, areaName(a))
			}
			sf.Name = sf.Name + "(" + strings.Join(names, "・") + ")"
			sf.Reference = code + "-" + areaSlug(g.areas[0])
			sf.ID = food.SystemID("seven-eleven", sf.Reference)
		}
		out = append(out, sf)
	}
	return out
}

func sameValues(a, b app.SystemFood) bool {
	if len(a.Nutrients) != len(b.Nutrients) || a.Basis.Unit != b.Basis.Unit {
		return false
	}
	for code, va := range a.Nutrients {
		vb, ok := b.Nutrients[code]
		if !ok || !va.Amount.Equal(vb.Amount) || va.Status != vb.Status {
			return false
		}
	}
	return true
}

func areaName(slug string) string {
	for _, a := range sevenAreas {
		if a.slug == slug {
			return a.name
		}
	}
	return "全国"
}

func areaSlug(slug string) string {
	if slug == "" {
		return "all"
	}
	return slug
}

func parseSeven(doc extract.Node, code string) (app.SystemFood, bool) {
	name := extract.Normalize(doc.Find("div", "item_ttl").Find("h1", "").Text())
	label := doc.Find("div", "allergy").Definitions()["栄養成分"]
	if name == "" || label == "" {
		return app.SystemFood{}, false
	}
	basis, found, v := extract.ParseLabel(label)
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	if !found {
		basis = food.Basis{Unit: "1食"}
	}
	return app.SystemFood{
		ID:        food.SystemID("seven-eleven", code),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "セブン‐イレブン",
		Reference: code,
		Nutrients: v.Profile(),
		Basis:     basis,
	}, true
}
