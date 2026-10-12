package official

import (
	"context"
	"regexp"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const lawsonBase = "https://www.lawson.co.jp"

// オリジナル商品の一覧と商品詳細, 値は関東地域
type Lawson struct {
	Fetch    *webfetch.Client
	Base     string
	MaxPages int
}

var _ port.Catalog = (*Lawson)(nil)

func (l *Lawson) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "lawson",
		Kind:        food.DatasetOfficial,
		Name:        "ローソン",
		License:     "lawson.co.jp terms of use (private use)",
		URL:         lawsonBase + "/recommend/original/",
		Attribution: "出典: ローソン オリジナル商品情報(lawson.co.jp)",
	}
}

var lawsonItem = regexp.MustCompile(`^/recommend/original/detail/(\d+_\d+)\.html$`)

func (l *Lawson) base() string {
	if l.Base != "" {
		return strings.TrimRight(l.Base, "/")
	}
	return lawsonBase
}

func (l *Lawson) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	maxPages := l.MaxPages
	if maxPages == 0 {
		maxPages = 500
	}
	c := crawler{
		fetch: l.Fetch,
		base:  l.base(),
		follow: func(p string) bool {
			return strings.HasPrefix(p, "/recommend/original/") && !strings.HasPrefix(p, "/recommend/original/detail/")
		},
		item:     lawsonItem,
		maxPages: maxPages,
		hosts:    []string{"www.lawson.co.jp"},
	}
	paths, err := c.collect(ctx, []string{"/recommend/original/select/index.html"})
	if err != nil {
		return prev, err
	}
	for _, path := range paths {
		code := lawsonItem.FindStringSubmatch(path)[1]
		ids := []food.ID{food.SystemID("lawson", code)}
		err := fetchDetail(ctx, l.Fetch, sink, ids, l.base()+path, func(doc catalog.Node) []app.SystemFood {
			sf, ok := parseLawson(doc, code)
			sf.URL = lawsonBase + path
			return single(sf, ok)
		})
		if err != nil {
			return prev, err
		}
	}
	return port.CatalogState{}, nil
}

var bracketBasis = regexp.MustCompile(`【(.+?)(?:当たり|あたり)】`)

func parseLawson(doc catalog.Node, code string) (app.SystemFood, bool) {
	detail := doc.Find("article", "productDetail")
	name := catalog.Normalize(detail.Find("h2", "ttl").Text())
	table := detail.Find("div", "nutritionFacts_table")
	if name == "" || !table.Ok() {
		return app.SystemFood{}, false
	}
	var v catalog.Values
	for key, val := range table.Definitions() {
		amount, unit := catalog.SplitValue(val)
		v.Set(key, amount, unit)
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	basis := food.Basis{Unit: "1食"}
	if m := bracketBasis.FindStringSubmatch(catalog.Normalize(table.Find("h3", "").Text())); m != nil {
		basis = catalog.BasisOf(m[1], "")
	}
	// 規格の重量は全量の単位, 1個等の包装単位は基準量の重量
	spec := catalog.Normalize(detail.Find("dl", "detail").Definitions()["規格(内容量)"])
	var portions []food.Portion
	if g, ok := catalog.Weight(spec); ok {
		switch {
		case !g.IsPositive():
		case !basis.PerUnit():
			portions = []food.Portion{{Unit: "全量", Grams: g}}
		case basis.Grams.IsZero() && strings.HasPrefix(basis.Unit, "1"):
			basis.Grams = g
		}
	}
	return app.SystemFood{
		ID:        food.SystemID("lawson", code),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "ローソン",
		Reference: code,
		Nutrients: v.Profile(),
		Basis:     basis,
		Portions:  portions,
	}, true
}
