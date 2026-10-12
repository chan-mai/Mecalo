package official

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const godivaBase = "https://www.godiva.co.jp"

// ドリンク一覧とサイズ別の商品詳細
type Godiva struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Godiva)(nil)

const godivaPath = "/drink/"

func (g *Godiva) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "godiva",
		Kind:        food.DatasetOfficial,
		Name:        "ゴディバ",
		License:     "godiva.co.jp terms of use (private use)",
		URL:         godivaBase + godivaPath,
		Attribution: "出典: ゴディバ ドリンク(godiva.co.jp)",
	}
}

var godivaItem = regexp.MustCompile(`^/drink/((?:clr_|cacaofruitjuice)[\w-]*)\.html$`)

// 1商品のサイズ数の上限
const godivaSizes = 3

func (g *Godiva) base() string {
	if g.Base != "" {
		return strings.TrimRight(g.Base, "/")
	}
	return godivaBase
}

func (g *Godiva) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	res, err := g.Fetch.Get(ctx, g.base()+godivaPath, webfetch.Validators{})
	if err != nil {
		return prev, err
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return prev, err
	}
	for _, slug := range godivaSlugs(doc) {
		if skipped, err := godivaSeenFresh(sink, slug); skipped || err != nil {
			if err != nil {
				return prev, err
			}
			continue
		}
		path := godivaPath + slug + ".html"
		res, err := g.Fetch.Get(ctx, g.base()+path, webfetch.Validators{})
		if errors.Is(err, webfetch.ErrNotFound) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return prev, err
			}
			if err := godivaFailed(sink, slug, err); err != nil {
				return prev, err
			}
			continue
		}
		doc, err := catalog.ParseHTML(res.Body)
		if err != nil {
			return prev, err
		}
		for _, sf := range parseGodiva(doc, slug) {
			sf.URL = godivaBase + path
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{}, nil
}

// 一覧本文内の商品ページ, 発見順
func godivaSlugs(doc catalog.Node) []string {
	list, _ := url.Parse(godivaBase + godivaPath)
	var out []string
	found := map[string]bool{}
	for _, a := range doc.Find("", "pane-main").FindAll("a", "") {
		u, err := list.Parse(strings.TrimSpace(a.Attr("href")))
		if err != nil || u.Host != list.Host {
			continue
		}
		m := godivaItem.FindStringSubmatch(u.Path)
		if m == nil || found[m[1]] {
			continue
		}
		found[m[1]] = true
		out = append(out, m[1])
	}
	return out
}

func godivaID(slug string, row int) food.ID {
	return food.SystemID("godiva", slug+"-"+strconv.Itoa(row))
}

// サイズ数上限までの候補IDを確認
func godivaSeenFresh(sink port.CatalogSink, slug string) (bool, error) {
	skipped := false
	for row := 1; row <= godivaSizes; row++ {
		if id := godivaID(slug, row); sink.Fresh(id) {
			skipped = true
			if err := sink.Seen(id); err != nil {
				return true, err
			}
		}
	}
	return skipped, nil
}

func godivaFailed(sink port.CatalogSink, slug string, err error) error {
	for row := 1; row <= godivaSizes; row++ {
		if e := sink.Failed(godivaID(slug, row), err); e != nil {
			return e
		}
	}
	return nil
}

// サイズ別の行ごとに1件
func parseGodiva(doc catalog.Node, slug string) []app.SystemFood {
	h2 := doc.Find("div", "clr_box").Find("h2", "")
	// 英語名のspanを除外
	name := catalog.Normalize(strings.TrimSuffix(h2.Text(), h2.Find("span", "").Text()))
	table := doc.Find("table", "tbl-chocolixir_s")
	if name == "" || !table.Ok() {
		return nil
	}
	energy := -1
	var out []app.SystemFood
	row := 0
	for _, tr := range table.FindAll("tr", "") {
		cells := tr.Cells()
		if len(cells) == 0 {
			continue
		}
		if cells[0].Data == "th" {
			for i, c := range cells {
				if strings.Contains(c.Text(), "カロリー") {
					energy = i
				}
			}
			continue
		}
		row++
		size := catalog.Normalize(cells[0].Text())
		if energy < 0 || energy >= len(cells) || size == "" || row > godivaSizes {
			continue
		}
		var v catalog.Values
		if amount, unit := splitValue(cells[energy].Text()); unit != "kcal" || !v.Set("エネルギー", amount, unit) {
			continue
		}
		ref := slug + "-" + strconv.Itoa(row)
		out = append(out, app.SystemFood{
			ID:        godivaID(slug, row),
			Kind:      food.KindProduct,
			Name:      name + " " + size,
			Brand:     "ゴディバ",
			Reference: ref,
			Nutrients: v.Profile(),
			Basis:     food.Basis{Unit: "1杯"},
		})
	}
	return out
}
