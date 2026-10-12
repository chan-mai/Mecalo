package official

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const doutorBase = "https://allergy.doutor.co.jp"

// ドトールグループの検索サイトの全件一覧とサイズ別の商品詳細
type Doutor struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Doutor)(nil)

type doutorBrand struct{ slug, name string }

// URLの識別子と表示名
var doutorBrands = []doutorBrand{
	{"doutor_coffee_shop", "ドトールコーヒーショップ"}, {"excelsior_caffe", "エクセルシオール カフェ"},
}

func (d *Doutor) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "doutor-group",
		Kind:        food.DatasetOfficial,
		Name:        "ドトール・エクセルシオール",
		License:     "doutor.co.jp terms of use (private use)",
		URL:         doutorBase + "/allergy_check/",
		Attribution: "出典: ドトールコーヒー アレルギー情報・栄養成分情報検索サイト",
	}
}

var doutorItem = regexp.MustCompile(`/allergy_check/(\w+)/item/detail/(\d+)/?$`)

// 1商品のサイズ数の上限
const doutorSizes = 4

func (d *Doutor) base() string {
	if d.Base != "" {
		return strings.TrimRight(d.Base, "/")
	}
	return doutorBase
}

func (d *Doutor) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	for _, brand := range doutorBrands {
		codes, err := d.list(ctx, brand.slug)
		if err != nil {
			return prev, err
		}
		for _, code := range codes {
			ref := brand.slug + "-" + code
			if skipped, err := doutorSeenFresh(sink, ref); skipped || err != nil {
				if err != nil {
					return prev, err
				}
				continue
			}
			path := "/allergy_check/" + brand.slug + "/item/detail/" + code
			res, err := d.Fetch.Get(ctx, d.base()+path, webfetch.Validators{})
			if errors.Is(err, webfetch.ErrNotFound) {
				continue
			}
			if err != nil {
				if ctx.Err() != nil {
					return prev, err
				}
				if err := doutorFailed(sink, ref, err); err != nil {
					return prev, err
				}
				continue
			}
			doc, err := catalog.ParseHTML(res.Body)
			if err != nil {
				return prev, err
			}
			for _, sf := range parseDoutor(doc, brand, ref) {
				sf.URL = doutorBase + path
				if err := sink.Emit(sf); err != nil {
					return prev, err
				}
			}
		}
	}
	return port.CatalogState{}, nil
}

// 全カテゴリの一覧の商品コード, 発見順
func (d *Doutor) list(ctx context.Context, slug string) ([]string, error) {
	res, err := d.Fetch.Get(ctx, d.base()+"/allergy_check/"+slug+"/search/?menu_category=all", webfetch.Validators{})
	if err != nil {
		return nil, err
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return nil, err
	}
	var codes []string
	found := map[string]bool{}
	for _, a := range doc.FindAll("a", "") {
		m := doutorItem.FindStringSubmatch(a.Attr("href"))
		if m != nil && m[1] == slug && !found[m[2]] {
			found[m[2]] = true
			codes = append(codes, m[2])
		}
	}
	return codes, nil
}

// サイズなしとサイズ数上限までの候補ID
func doutorIDs(ref string) []food.ID {
	ids := []food.ID{food.SystemID("doutor-group", ref)}
	for i := 1; i <= doutorSizes; i++ {
		ids = append(ids, food.SystemID("doutor-group", ref+"-"+strconv.Itoa(i)))
	}
	return ids
}

func doutorSeenFresh(sink port.CatalogSink, ref string) (bool, error) {
	skipped := false
	for _, id := range doutorIDs(ref) {
		if sink.Fresh(id) {
			skipped = true
			if err := sink.Seen(id); err != nil {
				return true, err
			}
		}
	}
	return skipped, nil
}

func doutorFailed(sink port.CatalogSink, ref string, err error) error {
	for _, id := range doutorIDs(ref) {
		if e := sink.Failed(id, err); e != nil {
			return e
		}
	}
	return nil
}

// サイズのタブごとに1件, タブなしは1件
func parseDoutor(doc catalog.Node, brand doutorBrand, ref string) []app.SystemFood {
	name := catalog.Normalize(doc.Find("h1", "detailHeader_title").Text())
	if name == "" {
		return nil
	}
	var sizes []string
	for _, li := range doc.Find("ul", "infoTab_btnsList").FindAll("li", "") {
		sizes = append(sizes, catalog.Normalize(li.Text()))
	}
	var out []app.SystemFood
	for i, block := range doc.FindAll("div", "infoTab_block") {
		var v catalog.Values
		for _, item := range block.FindAll("li", "infoTab_item") {
			amount, unit := splitValue(item.Find("p", "infoTab_txt").Text())
			v.Set(item.Find("p", "infoTab_title").Text(), amount, unit)
		}
		if v.Len() == 0 {
			continue
		}
		label, r := name, ref
		if len(sizes) > 0 {
			r = ref + "-" + strconv.Itoa(i+1)
			if i < len(sizes) && sizes[i] != "" {
				label = name + " " + sizes[i]
			}
		}
		out = append(out, app.SystemFood{
			ID:        food.SystemID("doutor-group", r),
			Kind:      food.KindProduct,
			Name:      label,
			Brand:     brand.name,
			Reference: r,
			Nutrients: v.Profile(),
			Basis:     food.Basis{Unit: "1食"},
		})
	}
	return out
}
