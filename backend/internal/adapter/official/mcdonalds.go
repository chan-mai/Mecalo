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

const mcdonaldsBase = "https://www.mcdonalds.co.jp"

// 全商品の栄養成分表1ページ
type McDonalds struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*McDonalds)(nil)

const mcdonaldsPath = "/quality/allergy_Nutrition/nutrient/"

func (m *McDonalds) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "mcdonalds",
		Kind:        food.DatasetOfficial,
		Name:        "マクドナルド",
		License:     "mcdonalds.co.jp terms of use (private use)",
		URL:         mcdonaldsBase + mcdonaldsPath,
		Attribution: "出典: 日本マクドナルド 栄養成分一覧",
	}
}

func (m *McDonalds) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(m.Base, "/")
	if base == "" {
		base = mcdonaldsBase
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, m.Fetch, []string{base + mcdonaldsPath}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	doc, err := catalog.ParseHTML(bodies[0])
	if err != nil {
		return prev, err
	}
	for _, sf := range parseMcDonalds(doc) {
		if err := sink.Emit(sf); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

var mcdonaldsProduct = regexp.MustCompile(`/products/(\d+)/?$`)

func parseMcDonalds(doc catalog.Node) []app.SystemFood {
	var out []app.SystemFood
	refs := map[string]bool{}
	for _, row := range nutritionRows(doc) {
		m := mcdonaldsProduct.FindStringSubmatch(row.href)
		if m == nil {
			continue
		}
		// サイズ違いは同じ商品ページ
		ref := m[1]
		if refs[ref] {
			ref = m[1] + "-" + catalog.Normalize(row.name)
		}
		refs[ref] = true
		out = append(out, app.SystemFood{
			ID:        food.SystemID("mcdonalds", ref),
			Kind:      food.KindProduct,
			Name:      row.name,
			Brand:     "マクドナルド",
			Reference: ref,
			Nutrients: row.values.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       mcdonaldsBase + "/products/" + m[1] + "/",
		})
	}
	return out
}

type nutritionRow struct {
	name   string
	href   string
	values catalog.Values
}

// 見出しにエネルギー等を含む表の各行, 1列目は商品名
func nutritionRows(doc catalog.Node) []nutritionRow {
	var out []nutritionRow
	for _, table := range doc.FindAll("table", "") {
		var heads []string
		for _, tr := range table.FindAll("tr", "") {
			if ths := tr.FindAll("th", ""); len(ths) > 1 {
				heads = heads[:0]
				for _, th := range ths {
					heads = append(heads, th.Text())
				}
			}
		}
		joined := strings.Join(heads, " ")
		if !strings.Contains(joined, "エネルギー") && !strings.Contains(joined, "熱量") {
			continue
		}
		for _, tr := range table.FindAll("tr", "") {
			cells := tr.Cells()
			if len(cells) < 2 || cells[1].Data != "td" {
				continue
			}
			// 商品名の見出しが別の行にある表は見出しが1列少ない
			offset := len(cells) - len(heads)
			if offset < 0 || offset > 1 {
				continue
			}
			name := catalog.Normalize(cells[0].Text())
			var v catalog.Values
			for i := 1; i < len(cells); i++ {
				h := i - offset
				if h < 0 || h >= len(heads) {
					continue
				}
				n, unit := splitHeading(heads[h])
				amount, valueUnit := splitValue(cells[i].Text())
				if valueUnit != "" {
					unit = valueUnit
				}
				v.Set(n, amount, unit)
			}
			if name == "" || v.Len() == 0 {
				continue
			}
			out = append(out, nutritionRow{name: name, href: cells[0].Find("a", "").Attr("href"), values: v})
		}
	}
	return out
}
