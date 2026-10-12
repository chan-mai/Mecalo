package official

import (
	"context"
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
		path := godivaPath + slug + ".html"
		err := fetchDetail(ctx, g.Fetch, sink, godivaIDs(slug), g.base()+path, func(doc catalog.Node) []app.SystemFood {
			out := parseGodiva(doc, slug)
			for i := range out {
				out[i].URL = godivaBase + path
			}
			return out
		})
		if err != nil {
			return prev, err
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

// サイズ数上限までの候補ID
func godivaIDs(slug string) []food.ID {
	ids := make([]food.ID, 0, godivaSizes)
	for row := 1; row <= godivaSizes; row++ {
		ids = append(ids, godivaID(slug, row))
	}
	return ids
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
		if amount, unit := catalog.SplitValue(cells[energy].Text()); unit != "kcal" || !v.Set("エネルギー", amount, unit) {
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
