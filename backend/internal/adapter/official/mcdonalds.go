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
	for _, row := range catalog.TableRows(doc) {
		m := mcdonaldsProduct.FindStringSubmatch(row.Href)
		if m == nil {
			continue
		}
		// サイズ違いは同じ商品ページ
		ref := m[1]
		if refs[ref] {
			ref = m[1] + "-" + catalog.Normalize(row.Name)
		}
		refs[ref] = true
		out = append(out, app.SystemFood{
			ID:        food.SystemID("mcdonalds", ref),
			Kind:      food.KindProduct,
			Name:      row.Name,
			Brand:     "マクドナルド",
			Reference: ref,
			Nutrients: row.Values.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       mcdonaldsBase + "/products/" + m[1] + "/",
		})
	}
	return out
}
