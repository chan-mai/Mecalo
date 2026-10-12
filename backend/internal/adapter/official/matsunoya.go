package official

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const matsunoyaBase = "https://www.matsuyafoods.co.jp"

// 特定原材料一覧に載るメニューの商品詳細, 値はみそ汁込みの店内量目
type Matsunoya struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Matsunoya)(nil)

func (m *Matsunoya) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "matsunoya",
		Kind:        food.DatasetOfficial,
		Name:        "松のや",
		License:     "matsuyafoods.co.jp site policy (private use)",
		URL:         matsunoyaBase + "/matsunoya/menu/",
		Attribution: "出典: 松のや メニュー情報(matsuyafoods.co.jp)",
	}
}

var matsunoyaItem = regexp.MustCompile(`^/matsunoya/menu/([a-z0-9_]+)/([A-Za-z0-9_-]+)\.html$`)

func (m *Matsunoya) base() string {
	if m.Base != "" {
		return strings.TrimRight(m.Base, "/")
	}
	return matsunoyaBase
}

func (m *Matsunoya) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	res, err := m.Fetch.Get(ctx, m.base()+"/matsunoya/safety/allergen.html", webfetch.Validators{})
	if err != nil {
		return prev, err
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return prev, err
	}
	c := crawler{hosts: []string{"www.matsuyafoods.co.jp"}}
	found := map[string]bool{}
	for _, cell := range doc.FindAll("td", "all_menu") {
		path := c.local(cell.Find("p", "al_title").Find("a", "").Attr("href"))
		item := matsunoyaItem.FindStringSubmatch(path)
		// 同じメニューが複数の分類に掲載
		if item == nil || found[item[2]] {
			continue
		}
		found[item[2]] = true
		if err := m.item(ctx, path, item[2], sink); err != nil {
			return prev, err
		}
	}
	return port.CatalogState{}, nil
}

func (m *Matsunoya) item(ctx context.Context, path, slug string, sink port.CatalogSink) error {
	id := food.SystemID("matsunoya", slug)
	if sink.Fresh(id) {
		return sink.Seen(id)
	}
	res, err := m.Fetch.Get(ctx, m.base()+path, webfetch.Validators{})
	if errors.Is(err, webfetch.ErrNotFound) {
		return nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		return sink.Failed(id, err)
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return err
	}
	sf, ok := parseMatsunoya(doc, slug)
	if !ok {
		return nil
	}
	sf.URL = matsunoyaBase + path
	return sink.Emit(sf)
}

// 「カロリー／1325kcal」形式の先頭の表示
func parseMatsunoya(doc catalog.Node, slug string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "ttl").Text())
	block := doc.Find("div", "nourishment")
	if name == "" || !block.Ok() {
		return app.SystemFood{}, false
	}
	_, _, v := catalog.ParseLabel(block.Find("p", "").Text())
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("matsunoya", slug),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "松のや",
		Reference: slug,
		Nutrients: v.Profile(),
		Basis:     food.Basis{Unit: "1食"},
	}, true
}
