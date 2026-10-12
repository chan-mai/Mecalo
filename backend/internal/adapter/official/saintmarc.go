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

const saintMarcBase = "https://www.saint-marc-hd.jp"

// サンマルクグループの検索サイトの全件一覧と商品詳細
type SaintMarc struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*SaintMarc)(nil)

type saintMarcBrand struct{ code, name string }

// 栄養成分を掲載するブランドのコードと表示名
var saintMarcBrands = []saintMarcBrand{
	{"04-O-A", "サンマルクカフェ"}, {"12-B-A", "倉式珈琲店"},
}

func (s *SaintMarc) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "saint-marc",
		Kind:        food.DatasetOfficial,
		Name:        "サンマルクグループ",
		License:     "saint-marc-hd.jp (terms not found)",
		URL:         saintMarcBase + "/allergy_check/",
		Attribution: "出典: サンマルクグループ アレルゲン検索・原産地表示サイト",
	}
}

var saintMarcItem = regexp.MustCompile(`/allergy_check/brand/([\w-]+)/detail/(\w+)/?$`)

func (s *SaintMarc) base() string {
	if s.Base != "" {
		return strings.TrimRight(s.Base, "/")
	}
	return saintMarcBase
}

func (s *SaintMarc) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	for _, brand := range saintMarcBrands {
		codes, err := s.list(ctx, brand.code)
		if err != nil {
			return prev, err
		}
		for _, code := range codes {
			// 商品コードはブランド間で重複
			ref := brand.code + "-" + code
			id := food.SystemID("saint-marc", ref)
			if sink.Fresh(id) {
				if err := sink.Seen(id); err != nil {
					return prev, err
				}
				continue
			}
			path := "/allergy_check/brand/" + brand.code + "/detail/" + code
			res, err := s.Fetch.Get(ctx, s.base()+path, webfetch.Validators{})
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
			if sf, ok := parseSaintMarc(doc, brand.name, ref); ok {
				sf.URL = saintMarcBase + path
				if err := sink.Emit(sf); err != nil {
					return prev, err
				}
			}
		}
	}
	return port.CatalogState{}, nil
}

// アレルゲン指定なしの検索結果の商品コード, 発見順
func (s *SaintMarc) list(ctx context.Context, brand string) ([]string, error) {
	res, err := s.Fetch.Get(ctx, s.base()+"/allergy_check/brand/"+brand+"/search/?allergy[]=&contain=0", webfetch.Validators{})
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
		m := saintMarcItem.FindStringSubmatch(a.Attr("href"))
		if m != nil && m[1] == brand && !found[m[2]] {
			found[m[2]] = true
			codes = append(codes, m[2])
		}
	}
	return codes, nil
}

func parseSaintMarc(doc catalog.Node, brand, ref string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "food_name").Text())
	var v catalog.Values
	for _, c := range doc.FindAll("li", "food_component") {
		amount, unit := catalog.SplitValue(c.Find("p", "food_componentNum").Text())
		v.Set(c.Find("p", "food_componentName").Text(), amount, unit)
	}
	if name == "" || v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("saint-marc", ref),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     brand,
		Reference: ref,
		Nutrients: v.Profile(),
		Basis:     food.Basis{Unit: "1食"},
	}, true
}
