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

const familymartBase = "https://www.family.co.jp"

// 中食・FFの栄養成分一覧ページ
type FamilyMart struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*FamilyMart)(nil)

func (m *FamilyMart) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "familymart",
		Kind:        food.DatasetOfficial,
		Name:        "ファミリーマート",
		License:     "family.co.jp terms of use (private use)",
		URL:         familymartBase + "/goods/safety.html",
		Attribution: "出典: ファミリーマート 商品の栄養成分・アレルゲン情報(family.co.jp)",
	}
}

func (m *FamilyMart) base() string {
	if m.Base != "" {
		return strings.TrimRight(m.Base, "/")
	}
	return familymartBase
}

var (
	familymartPage  = regexp.MustCompile(`/goods/safety/[a-z0-9]+\.html$`)
	familymartItem  = regexp.MustCompile(`/goods/[a-z0-9_]+/(\d+)\.html$`)
	familymartImage = regexp.MustCompile(`/goods/(\d{5,})\.(?:jpg|png)$`)
)

func (m *FamilyMart) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	idx, err := m.Fetch.Get(ctx, m.base()+"/goods/safety.html", webfetch.Validators{})
	if err != nil {
		return prev, err
	}
	doc, err := catalog.ParseHTML(idx.Body)
	if err != nil {
		return prev, err
	}
	var urls []string
	seen := map[string]bool{}
	for _, a := range doc.FindAll("a", "") {
		href := a.Attr("href")
		loc := familymartPage.FindString(href)
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true
		urls = append(urls, m.base()+loc)
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, m.Fetch, urls, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	emitted := map[food.ID]bool{}
	for _, body := range bodies {
		page, err := catalog.ParseHTML(body)
		if err != nil {
			return prev, err
		}
		for _, sf := range parseFamilyMart(page) {
			if emitted[sf.ID] {
				continue
			}
			emitted[sf.ID] = true
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

func parseFamilyMart(page catalog.Node) []app.SystemFood {
	var out []app.SystemFood
	for _, li := range page.FindAll("li", "") {
		info := li.Find("div", "item_basic_info")
		table := li.Find("table", "item_nutritional_info")
		if !info.Ok() || !table.Ok() {
			continue
		}
		title := info.Find("p", "name")
		name := catalog.Normalize(title.Text())
		code, url := "", ""
		if m := familymartItem.FindStringSubmatch(title.Find("a", "").Attr("href")); m != nil {
			code, url = m[1], familymartBase+m[0]
		} else if m := familymartImage.FindStringSubmatch(li.Find("img", "").Attr("src")); m != nil {
			// 商品ページなしは画像の商品コード
			code = m[1]
		}
		if code == "" || name == "" {
			continue
		}
		var v catalog.Values
		heads := table.FindAll("th", "")
		cells := table.FindAll("td", "")
		for i := 0; i < len(heads) && i < len(cells); i++ {
			name, unit := catalog.SplitHeading(heads[i].Text())
			v.Set(name, cells[i].Text(), unit)
		}
		if v.Len() == 0 {
			continue
		}
		out = append(out, app.SystemFood{
			ID:        food.SystemID("familymart", code),
			Kind:      food.KindProduct,
			Name:      name,
			Brand:     "ファミリーマート",
			Reference: code,
			Nutrients: v.Profile(),
			Basis:     food.Basis{Unit: "1食"},
			URL:       url,
		})
	}
	return out
}
