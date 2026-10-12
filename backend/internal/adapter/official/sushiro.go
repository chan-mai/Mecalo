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

const sushiroBase = "https://www.akindo-sushiro.co.jp"

// 地域ごとの代表店舗の店舗別メニュー
type Sushiro struct {
	Fetch *webfetch.Client
	Base  string
}

var _ port.Catalog = (*Sushiro)(nil)

func (s *Sushiro) Dataset() food.Dataset {
	return food.Dataset{
		ID:          "sushiro",
		Kind:        food.DatasetOfficial,
		Name:        "スシロー",
		License:     "akindo-sushiro.co.jp terms of use (private use)",
		URL:         sushiroBase + "/menu/",
		Attribution: "出典: スシロー メニュー(akindo-sushiro.co.jp)",
	}
}

// 地域の代表都道府県コード, 表示順
var sushiroPrefs = []string{"01", "04", "13", "23", "27", "34", "37", "40", "47"}

func (s *Sushiro) base() string {
	if s.Base != "" {
		return strings.TrimRight(s.Base, "/")
	}
	return sushiroBase
}

func (s *Sushiro) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	var paths, urls []string
	for _, pref := range sushiroPrefs {
		id, err := s.store(ctx, pref)
		if err != nil {
			return prev, err
		}
		if id != "" {
			paths = append(paths, "/menu/menu_detail/?s_id="+id)
			urls = append(urls, s.base()+paths[len(paths)-1])
		}
	}
	if len(urls) == 0 {
		return prev, errors.New("sushiro: no stores found")
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, s.Fetch, urls, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	refs := map[string]bool{}
	for i, body := range bodies {
		doc, err := catalog.ParseHTML(body)
		if err != nil {
			return prev, err
		}
		for _, sf := range parseSushiro(doc) {
			if refs[sf.Reference] {
				continue
			}
			refs[sf.Reference] = true
			sf.URL = sushiroBase + paths[i]
			if err := sink.Emit(sf); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

var sushiroItems = regexp.MustCompile(`(?m)^\s*items = (\[.+\]);\s*$`)

// 都道府県の最初の店舗ID, 店舗なしは空
func (s *Sushiro) store(ctx context.Context, pref string) (string, error) {
	res, err := s.Fetch.Get(ctx, s.base()+"/menu/?pref="+pref+"&mode=service", webfetch.Validators{})
	if errors.Is(err, webfetch.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	m := sushiroItems.FindSubmatch(res.Body)
	if m == nil {
		return "", nil
	}
	var stores []struct {
		ID string `json:"sushipass_id"`
	}
	if err := json.Unmarshal(m[1], &stores); err != nil {
		return "", fmt.Errorf("sushiro: stores of pref %s: %w", pref, err)
	}
	for _, st := range stores {
		if st.ID != "" {
			return st.ID, nil
		}
	}
	return "", nil
}

// 「53kcal」「100mlあたり48kcal」
var sushiroCalorie = regexp.MustCompile(`^(?:(.+?)(?:当たり|あたり))?\s*([\d.,]+)\s*kcal$`)

func parseSushiro(doc catalog.Node) []app.SystemFood {
	var out []app.SystemFood
	for _, item := range doc.FindAll("div", "menu-item") {
		// モーダル側はdata-targetなし
		ref := item.Attr("data-target")
		name := catalog.Normalize(item.Find("div", "menu-item__name").Text())
		m := sushiroCalorie.FindStringSubmatch(catalog.Normalize(item.Find("div", "menu-item__calorie").Text()))
		if ref == "" || name == "" || m == nil {
			continue
		}
		var v catalog.Values
		if !v.Set("エネルギー", m[2], "kcal") {
			continue
		}
		out = append(out, app.SystemFood{
			ID:        food.SystemID("sushiro", ref),
			Kind:      food.KindProduct,
			Name:      name,
			Brand:     "スシロー",
			Reference: ref,
			Nutrients: v.Profile(),
			Basis:     catalog.BasisOf(m[1], ""),
		})
	}
	return out
}
