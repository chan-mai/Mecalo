package official

import (
	"context"
	"encoding/json"
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

const kiyokenBase = "https://kiyoken.com"

// WordPressの商品一覧と商品詳細, 栄養表示のない商品は対象外
type Kiyoken struct {
	Fetch *webfetch.Client
	Base  string
	// 一覧ページ数の上限, 0は無制限
	MaxPages int
}

var _ port.Catalog = (*Kiyoken)(nil)

func (k *Kiyoken) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "kiyoken",
		Kind:        food.DatasetOfficial,
		Name:        "崎陽軒",
		License:     "kiyoken.com terms of use (private use)",
		URL:         kiyokenBase + "/products/",
		Attribution: "出典: 崎陽軒 商品紹介(kiyoken.com)",
	}
}

const kiyokenPerPage = 100

var kiyokenItem = regexp.MustCompile(`^/products/([a-z0-9_-]+)/([A-Za-z0-9_-]+)\.html$`)

func (k *Kiyoken) base() string {
	if k.Base != "" {
		return strings.TrimRight(k.Base, "/")
	}
	return kiyokenBase
}

func (k *Kiyoken) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	c := crawler{hosts: []string{"kiyoken.com"}}
	found := map[string]bool{}
	for page := 1; k.MaxPages == 0 || page <= k.MaxPages; page++ {
		res, err := k.Fetch.Get(ctx, fmt.Sprintf("%s/wp-json/wp/v2/products?per_page=%d&page=%d", k.base(), kiyokenPerPage, page), webfetch.Validators{})
		// 最終ページの次
		if page > 1 && errors.Is(err, webfetch.ErrNotFound) {
			break
		}
		if err != nil {
			return prev, err
		}
		var posts []struct {
			Link string `json:"link"`
		}
		if err := json.Unmarshal(res.Body, &posts); err != nil {
			return prev, err
		}
		for _, p := range posts {
			path := c.local(p.Link)
			m := kiyokenItem.FindStringSubmatch(path)
			if m == nil || found[m[2]] {
				continue
			}
			found[m[2]] = true
			if err := k.item(ctx, path, m[2], sink); err != nil {
				return prev, err
			}
		}
		if len(posts) < kiyokenPerPage {
			break
		}
	}
	return port.CatalogState{}, nil
}

func (k *Kiyoken) item(ctx context.Context, path, slug string, sink port.CatalogSink) error {
	id := food.SystemID("kiyoken", slug)
	if sink.Fresh(id) {
		return sink.Seen(id)
	}
	res, err := k.Fetch.Get(ctx, k.base()+path, webfetch.Validators{})
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
	sf, ok := parseKiyoken(doc, slug)
	if !ok {
		return nil
	}
	sf.URL = kiyokenBase + path
	return sink.Emit(sf)
}

func parseKiyoken(doc catalog.Node, slug string) (app.SystemFood, bool) {
	name := catalog.Normalize(doc.Find("h1", "p-product__name").Text())
	label := doc.Find("table", "p-product__spec").Definitions()["栄養成分表示"]
	if name == "" || label == "" {
		return app.SystemFood{}, false
	}
	basis, found, v := catalog.ParseLabel(label)
	if !found {
		basis = food.Basis{Unit: "1食"}
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	return app.SystemFood{
		ID:        food.SystemID("kiyoken", slug),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     "崎陽軒",
		Reference: slug,
		Nutrients: v.Profile(),
		Basis:     basis,
	}, true
}
