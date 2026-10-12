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

const kourakuenBase = "https://stores.kourakuen.co.jp"

// グランドメニューの一覧と商品詳細
type Kourakuen struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Kourakuen)(nil)

func (k *Kourakuen) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "kourakuen",
		Kind:        food.DatasetOfficial,
		Name:        "幸楽苑",
		License:     "kourakuen.co.jp terms of service (private use)",
		URL:         kourakuenBase + "/grandmenu/",
		Attribution: "出典: 幸楽苑 グランドメニュー(stores.kourakuen.co.jp)",
	}
}

// 全件とカテゴリ別の一覧
var kourakuenLists = []string{"", "?gid=01", "?gid=02", "?gid=03", "?gid=04", "?gid=05", "?gid=06"}

var kourakuenItem = regexp.MustCompile(`info\.html\?id=(\d+)`)

func (k *Kourakuen) base() string {
	if k.Base != "" {
		return strings.TrimRight(k.Base, "/")
	}
	return kourakuenBase
}

func (k *Kourakuen) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	var codes []string
	found := map[string]bool{}
	for _, q := range kourakuenLists {
		res, err := k.Fetch.Get(ctx, k.base()+"/grandmenu/"+q, webfetch.Validators{})
		if q != "" && errors.Is(err, webfetch.ErrNotFound) {
			continue
		}
		if err != nil {
			return prev, err
		}
		doc, err := catalog.ParseHTML(res.Body)
		if err != nil {
			return prev, err
		}
		for _, a := range doc.FindAll("a", "") {
			m := kourakuenItem.FindStringSubmatch(a.Attr("href"))
			if m != nil && !found[m[1]] {
				found[m[1]] = true
				codes = append(codes, m[1])
			}
		}
	}
	for _, code := range codes {
		id := food.SystemID("kourakuen", code)
		if sink.Fresh(id) {
			if err := sink.Seen(id); err != nil {
				return prev, err
			}
			continue
		}
		path := "/grandmenu/info.html?id=" + code
		res, err := k.Fetch.Get(ctx, k.base()+path, webfetch.Validators{})
		if errors.Is(err, webfetch.ErrNotFound) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return prev, err
			}
			if err := sink.Failed(id, err); err != nil {
				return prev, err
			}
			continue
		}
		doc, err := catalog.ParseHTML(res.Body)
		if err != nil {
			return prev, err
		}
		if sf, ok := parseKourakuen(doc, code); ok {
			sf.URL = kourakuenBase + path
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{}, nil
}

func parseKourakuen(doc catalog.Node, code string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "menuMainTitle").Text())
	facts := doc.Find("div", "nutritionFacts")
	if name == "" || !facts.Ok() {
		return app.SystemFood{}, false
	}
	var v catalog.Values
	for key, val := range facts.Definitions() {
		// 「639Kcal」表記
		amount, unit := catalog.SplitValue(strings.ToLower(val))
		v.Set(key, amount, unit)
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("kourakuen", code),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "幸楽苑",
		Reference: code,
		Nutrients: v.Profile(),
		Basis:     food.Basis{Unit: "1食"},
	}, true
}
