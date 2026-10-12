// メーカー・小売・外食チェーン公式サイトの商品一覧同期
package official

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const topvaluBase = "https://www.topvalu.net"

// 食品カテゴリの一覧と商品詳細
type Topvalu struct {
	Fetch *webfetch.Client
	Base  string
	// 一覧ページ数の上限, 0は無制限
	MaxPages int
}

var _ port.Catalog = (*Topvalu)(nil)

func (t *Topvalu) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "topvalu",
		Kind:        food.DatasetOfficial,
		Name:        "トップバリュ",
		License:     "topvalu.net terms of use (private use)",
		URL:         topvaluBase + "/",
		Attribution: "出典: トップバリュ商品情報(topvalu.net)",
	}
}

var topvaluItem = regexp.MustCompile(`^/items/detail/(\d{8,14})$`)

func (t *Topvalu) base() string {
	if t.Base != "" {
		return strings.TrimRight(t.Base, "/")
	}
	return topvaluBase
}

func (t *Topvalu) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	seen := map[string]bool{}
	for page := 1; t.MaxPages == 0 || page <= t.MaxPages; page++ {
		res, err := t.Fetch.Get(ctx, fmt.Sprintf("%s/items/list/100000000?page=%d", t.base(), page), webfetch.Validators{})
		// 最終ページの次
		if page > 1 && errors.Is(err, webfetch.ErrNotFound) {
			break
		}
		if err != nil {
			return prev, err
		}
		doc, err := catalog.ParseHTML(res.Body)
		if err != nil {
			return prev, err
		}
		added := 0
		for _, a := range doc.FindAll("a", "") {
			m := topvaluItem.FindStringSubmatch(a.Attr("href"))
			if m == nil || seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			added++
			err := t.item(ctx, m[1], sink)
			if err != nil && !errors.Is(err, webfetch.ErrNotFound) {
				if ctx.Err() != nil {
					return prev, err
				}
				if err := sink.Failed(food.SystemID("topvalu", m[1]), err); err != nil {
					return prev, err
				}
			}
		}
		if added == 0 {
			break
		}
	}
	return port.CatalogState{}, nil
}

func (t *Topvalu) item(ctx context.Context, jan string, sink port.CatalogSink) error {
	id := food.SystemID("topvalu", jan)
	if sink.Fresh(id) {
		return sink.Seen(id)
	}
	url := t.base() + "/items/detail/" + jan
	res, err := t.Fetch.Get(ctx, url, webfetch.Validators{})
	if err != nil {
		return err
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return err
	}
	sf, ok := parseTopvalu(doc, jan)
	if !ok {
		return nil
	}
	sf.URL = topvaluBase + "/items/detail/" + jan
	return sink.Emit(sf)
}

func parseTopvalu(doc catalog.Node, jan string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "item-detail__name").Text())
	defs := doc.Definitions()
	label := defs["栄養成分"]
	if name == "" || label == "" {
		return app.SystemFood{}, false
	}
	basis, found, values := catalog.ParseLabel(label)
	if !found {
		basis = food.Basis{Unit: "1食"}
	}
	if values.Len() == 0 {
		return app.SystemFood{}, false
	}
	sf := app.SystemFood{
		ID:        food.SystemID("topvalu", jan),
		Kind:      food.KindProduct,
		Barcode:   jan,
		Name:      name,
		Brand:     "トップバリュ",
		Reference: jan,
		Nutrients: values.Profile(),
		Basis:     basis,
	}
	// 規格が重量のみの場合は全量の単位
	if g, ok := catalog.Weight(defs["規格"]); ok && (!basis.PerUnit() || basis.Grams.IsPositive()) {
		if g.IsPositive() && (!basis.PerUnit() || !basis.Grams.Equal(g)) {
			sf.Portions = []food.Portion{{Unit: "全量", Grams: g}}
		}
	}
	return sf, true
}
