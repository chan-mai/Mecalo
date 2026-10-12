package official

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const cozycornerBase = "https://www.cozycorner.co.jp"

// 商品紹介のカテゴリ一覧と商品詳細
type CozyCorner struct {
	Fetch    *webfetch.Client
	Base     string
	MaxPages int
}

var _ port.Catalog = (*CozyCorner)(nil)

func (c *CozyCorner) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "cozycorner",
		Kind:        food.DatasetOfficial,
		Name:        "銀座コージーコーナー",
		License:     "cozycorner.co.jp terms of use (private use)",
		URL:         cozycornerBase + "/product/",
		Attribution: "出典: 銀座コージーコーナー 商品紹介(cozycorner.co.jp)",
	}
}

var (
	cozycornerItem     = regexp.MustCompile(`^/product/[a-z0-9_]+/(\d+)\.html$`)
	cozycornerCategory = regexp.MustCompile(`^/product/[a-z0-9_]+/(?:index\.html)?$`)
)

func (c *CozyCorner) base() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/")
	}
	return cozycornerBase
}

func (c *CozyCorner) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	maxPages := c.MaxPages
	if maxPages == 0 {
		maxPages = 100
	}
	cr := crawler{
		fetch:    c.Fetch,
		base:     c.base(),
		follow:   cozycornerCategory.MatchString,
		item:     cozycornerItem,
		maxPages: maxPages,
		hosts:    []string{"www.cozycorner.co.jp"},
	}
	paths, err := cr.collect(ctx, []string{"/product/"})
	if err != nil {
		return prev, err
	}
	found := map[string]bool{}
	for _, path := range paths {
		code := cozycornerItem.FindStringSubmatch(path)[1]
		// 同じ商品が複数カテゴリに掲載
		if found[code] {
			continue
		}
		found[code] = true
		ids := []food.ID{food.SystemID("cozycorner", code)}
		err := fetchDetail(ctx, c.Fetch, sink, ids, c.base()+path, func(doc extract.Node) []app.SystemFood {
			sf, ok := parseCozyCorner(doc, code)
			sf.URL = cozycornerBase + path
			return single(sf, ok)
		})
		if err != nil {
			return prev, err
		}
	}
	return port.CatalogState{}, nil
}

var cozycornerBasis = regexp.MustCompile(`\((.+?)(?:当たり|あたり)\)`)

// PC用と同値のスマートフォン用の表
func parseCozyCorner(doc extract.Node, code string) (app.SystemFood, bool) {
	name := extract.Normalize(doc.Find("h1", "en").Text())
	var table extract.Node
	for _, t := range doc.FindAll("table", "nutrition") {
		if slices.Contains(strings.Fields(t.Attr("class")), "sp-only") {
			table = t
			break
		}
	}
	if name == "" || !table.Ok() {
		return app.SystemFood{}, false
	}
	var v extract.Values
	basis := food.Basis{Unit: "1食"}
	for _, row := range table.FindAll("tr", "") {
		cells := row.Cells()
		if len(cells) == 1 {
			if m := cozycornerBasis.FindStringSubmatch(extract.Normalize(cells[0].Text())); m != nil {
				basis = extract.BasisOf(m[1], "")
			}
			continue
		}
		if len(cells) == 2 {
			amount, unit := extract.SplitValue(cells[1].Text())
			v.Set(cells[0].Text(), amount, unit)
		}
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("cozycorner", code),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "銀座コージーコーナー",
		Reference: code,
		Nutrients: v.Profile(),
		Basis:     basis,
	}, true
}
