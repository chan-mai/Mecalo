package official

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const horai551Base = "https://www.551horai.co.jp"

// 商品案内の一覧と商品詳細
type Horai551 struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Horai551)(nil)

func (h *Horai551) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "horai551",
		Kind:        food.DatasetOfficial,
		Name:        "551蓬莱",
		License:     "551horai.co.jp (terms not found)",
		URL:         horai551Base + "/menu/",
		Attribution: "出典: 551蓬莱 商品案内(551horai.co.jp)",
	}
}

// 日本語のスラッグを含む, パスは復号済み
var horai551Item = regexp.MustCompile(`^/menu/([^/]+)/$`)

func (h *Horai551) base() string {
	if h.Base != "" {
		return strings.TrimRight(h.Base, "/")
	}
	return horai551Base
}

func (h *Horai551) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	c := crawler{
		fetch:    h.Fetch,
		base:     h.base(),
		follow:   func(string) bool { return false },
		item:     horai551Item,
		maxPages: 1,
		hosts:    []string{"www.551horai.co.jp"},
	}
	paths, err := c.collect(ctx, []string{"/menu/"})
	if err != nil {
		return prev, err
	}
	for _, path := range paths {
		slug := horai551Item.FindStringSubmatch(path)[1]
		id := food.SystemID("horai551", slug)
		if sink.Fresh(id) {
			if err := sink.Seen(id); err != nil {
				return prev, err
			}
			continue
		}
		escaped := (&url.URL{Path: path}).EscapedPath()
		res, err := h.Fetch.Get(ctx, h.base()+escaped, webfetch.Validators{})
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
		if sf, ok := parseHorai551(doc, slug); ok {
			sf.URL = horai551Base + escaped
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{}, nil
}

var horai551Unit = regexp.MustCompile(`\((.+?)\)`)

// 見出し行と値の行の表, 複数行は先頭のみ
func parseHorai551(doc catalog.Node, slug string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "").Text())
	rows := doc.Find("table", "nutritional").FindAll("tr", "")
	if name == "" || len(rows) < 2 {
		return app.SystemFood{}, false
	}
	heads, cells := rows[0].Cells(), rows[1].Cells()
	var v catalog.Values
	unit, grams := "1個", ""
	for i, head := range heads {
		if i >= len(cells) {
			break
		}
		key := strings.ReplaceAll(catalog.Normalize(head.Text()), " ", "")
		amount, u := catalog.SplitValue(cells[i].Text())
		if strings.HasPrefix(key, "量目") {
			if m := horai551Unit.FindStringSubmatch(key); m != nil {
				unit = m[1]
			}
			if u == "g" {
				grams = amount
			}
			continue
		}
		v.Set(key, amount, u)
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("horai551", slug),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "551蓬莱",
		Reference: slug,
		Nutrients: v.Profile(),
		Basis:     catalog.BasisOf(unit, grams),
	}, true
}
