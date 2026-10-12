package official

import (
	"context"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 全商品の栄養成分表が1ページにある取得元
type TablePage struct {
	Fetch *webfetch.Client
	// テスト用の接続先, 空はOrigin
	Base    string
	ID      food.DatasetID
	Name    string
	Origin  string
	Path    string
	License string
}

var _ port.Catalog = (*TablePage)(nil)

func (t *TablePage) Dataset() food.Dataset {
	return food.Dataset{
		ID:          t.ID,
		Kind:        food.DatasetOfficial,
		Name:        t.Name,
		License:     t.License,
		URL:         t.Origin + t.Path,
		Attribution: "出典: " + t.Name + " 栄養成分一覧",
	}
}

func (t *TablePage) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	base := strings.TrimRight(t.Base, "/")
	if base == "" {
		base = t.Origin
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, t.Fetch, []string{base + t.Path}, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	doc, err := extract.ParseHTML(bodies[0])
	if err != nil {
		return prev, err
	}
	for _, sf := range t.parse(doc) {
		if err := sink.Emit(sf); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

// 商品IDがないため正規化した商品名を識別子に
func (t *TablePage) parse(doc extract.Node) []app.SystemFood {
	var out []app.SystemFood
	refs := map[string]bool{}
	for _, row := range extract.TableRows(doc) {
		ref := food.SearchKey(row.Name)
		if refs[ref] {
			continue
		}
		refs[ref] = true
		out = append(out, app.SystemFood{
			ID:        food.SystemID(string(t.ID), ref),
			Kind:      food.KindProduct,
			Name:      row.Name,
			Brand:     t.Name,
			Reference: ref,
			Nutrients: row.Values.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       t.Origin + t.Path,
		})
	}
	return out
}

func RingerHut(f *webfetch.Client) *TablePage {
	return &TablePage{
		Fetch: f, ID: "ringerhut", Name: "リンガーハット", Origin: "https://www.ringerhut.jp",
		Path: "/quality/allergy-nutrition_value/", License: "ringerhut.jp site rules (private use)",
	}
}

// 店舗ごとの一覧のうち代表店舗の値
func Ootoya(f *webfetch.Client) *TablePage {
	return &TablePage{
		Fetch: f, ID: "ootoya", Name: "大戸屋", Origin: "https://www.ootoya.com",
		Path: "/menu_list/info/nutrition/27186", License: "ootoya.com (terms not found)",
	}
}
