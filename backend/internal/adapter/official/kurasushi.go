package official

import (
	"context"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const kuraSushiBase = "https://www.kurasushi.co.jp"

// 全商品のメニュー1ページ
type KuraSushi struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*KuraSushi)(nil)

const kuraSushiPath = "/menu/"

func (k *KuraSushi) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "kurasushi",
		Kind:        food.DatasetOfficial,
		Name:        "くら寿司",
		License:     "kurasushi.co.jp terms of use (private use)",
		URL:         kuraSushiBase + kuraSushiPath,
		Attribution: "出典: くら寿司 メニュー(kurasushi.co.jp)",
	}
}

func (k *KuraSushi) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(k.Base, "/")
	if base == "" {
		base = kuraSushiBase
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, k.Fetch, []string{base + kuraSushiPath}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	doc, err := catalog.ParseHTML(bodies[0])
	if err != nil {
		return prev, err
	}
	for _, sf := range parseKuraSushi(doc) {
		if err := sink.Emit(sf); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

func parseKuraSushi(doc catalog.Node) []app.SystemFood {
	var out []app.SystemFood
	refs := map[string]bool{}
	for _, item := range doc.FindAll("div", "menu-item") {
		ref, ok := strings.CutPrefix(item.Attr("id"), "menu-")
		name := catalog.Normalize(item.Find("h4", "menu-name").Text())
		if !ok || ref == "" || name == "" || refs[ref] {
			continue
		}
		// 1項目目に価格とエネルギー
		var v catalog.Values
		for _, p := range item.Find("ul", "menu-summary").Find("li", "").FindAll("p", "") {
			if amount, unit := splitValue(p.Text()); unit == "kcal" {
				v.Set("エネルギー", amount, unit)
			}
		}
		if v.Len() == 0 {
			continue
		}
		refs[ref] = true
		out = append(out, app.SystemFood{
			ID:        food.SystemID("kurasushi", ref),
			Kind:      food.KindProduct,
			Name:      name,
			Brand:     "くら寿司",
			Reference: ref,
			Nutrients: v.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       kuraSushiBase + kuraSushiPath,
		})
	}
	return out
}
