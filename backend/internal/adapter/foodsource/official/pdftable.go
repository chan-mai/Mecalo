package official

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 栄養成分表のPDFを公開する取得元
type PDFTable struct {
	Fetch *webfetch.Client
	// テスト用の接続先, 空はOrigin
	Base    string
	ID      food.DatasetID
	Name    string
	Origin  string
	License string
	// PDFのパス, 複数可
	Paths []string
	// PDFのURLに日付等が入る場合のリンク元ページ
	LinkPage string
	// リンク元ページ内のPDFのリンク
	LinkPattern *regexp.Regexp
	// 基準量, 空は1食
	Unit string
	// APIでURLを得る取得元, 指定時はPaths等より優先
	Resolve func(ctx context.Context, t *PDFTable) ([]string, error)
}

var _ port.Catalog = (*PDFTable)(nil)

func (t *PDFTable) Dataset() food.Dataset {
	return food.Dataset{
		ID:          t.ID,
		Kind:        food.DatasetOfficial,
		Name:        t.Name,
		License:     t.License,
		URL:         t.Origin + t.LinkPage,
		Attribution: "出典: " + t.Name + " 栄養成分一覧",
	}
}

func (t *PDFTable) base() string {
	if t.Base != "" {
		return strings.TrimRight(t.Base, "/")
	}
	return t.Origin
}

func (t *PDFTable) Sync(ctx context.Context, prev port.CatalogState, sink port.CatalogSink) (port.CatalogState, error) {
	urls, err := t.pdfURLs(ctx)
	if err != nil {
		return prev, err
	}
	bodies, cursor, unchanged, err := fetchPages(ctx, t.Fetch, urls, prev.Cursor)
	if err != nil || unchanged {
		return port.CatalogState{Cursor: cursor, Unchanged: unchanged}, err
	}
	seen := map[string]bool{}
	for _, body := range bodies {
		pages, err := extract.ReadPDF(body)
		if err != nil {
			return prev, err
		}
		for _, row := range extract.PDFRows(pages) {
			ref := food.SearchKey(row.Name)
			if seen[ref] {
				continue
			}
			seen[ref] = true
			unit := t.Unit
			if unit == "" {
				unit = "1食"
			}
			if err := sink.Emit(app.SystemFood{
				ID:        food.SystemID(string(t.ID), ref),
				Kind:      food.KindProduct,
				Name:      row.Name,
				Brand:     t.Name,
				Reference: ref,
				Nutrients: row.Values.Profile(),
				Basis:     food.Basis{Unit: unit},
				URL:       t.Origin + t.LinkPage,
			}); err != nil {
				return prev, err
			}
		}
	}
	return port.CatalogState{Cursor: cursor}, nil
}

// リンク元ページから最新のPDFのURLを解決
func (t *PDFTable) pdfURLs(ctx context.Context) ([]string, error) {
	if t.Resolve != nil {
		return t.Resolve(ctx, t)
	}
	if t.LinkPattern == nil {
		urls := make([]string, 0, len(t.Paths))
		for _, p := range t.Paths {
			urls = append(urls, t.base()+p)
		}
		return urls, nil
	}
	res, err := t.Fetch.Get(ctx, t.base()+t.LinkPage, webfetch.Validators{})
	if err != nil {
		return nil, err
	}
	doc, err := extract.ParseHTML(res.Body)
	if err != nil {
		return nil, err
	}
	page, err := url.Parse(t.base() + t.LinkPage)
	if err != nil {
		return nil, err
	}
	var urls []string
	seen := map[string]bool{}
	for _, a := range doc.FindAll("a", "") {
		href := strings.TrimSpace(a.Attr("href"))
		if !t.LinkPattern.MatchString(href) {
			continue
		}
		u, err := page.Parse(href)
		if err != nil || seen[u.String()] {
			continue
		}
		seen[u.String()] = true
		urls = append(urls, u.String())
	}
	if len(urls) == 0 {
		return nil, errors.New("pdf link not found on " + t.LinkPage)
	}
	return urls, nil
}

func Dennys(f *webfetch.Client) *PDFTable {
	return &PDFTable{
		Fetch: f, ID: "dennys", Name: "デニーズ", Origin: "https://www.dennys.jp",
		LinkPage: "/safety/nutritional/", Paths: []string{"/safety/pdf/nutritive_value_A.pdf"},
		License: "dennys.jp (terms not found)",
	}
}

func SteakDon(f *webfetch.Client) *PDFTable {
	paths := make([]string, 0, 5)
	for _, k := range []string{"st", "lunch", "side", "gentei", "to"} {
		paths = append(paths, "/pdf/allergy_don_"+k+".pdf")
	}
	return &PDFTable{
		Fetch: f, ID: "steakdon", Name: "ステーキのどん", Origin: "https://www.steak-don.jp",
		Paths: paths, License: "steak-don.jp (terms not found)",
	}
}

func Tonden(f *webfetch.Client) *PDFTable {
	return &PDFTable{
		Fetch: f, ID: "tonden", Name: "とんでん", Origin: "https://assets.tonden.co.jp",
		Paths: []string{"/allergie_list/allergie_list.pdf"}, License: "tonden.co.jp (terms not found)",
	}
}

// 最新PDFのURLを商品情報APIから取得
func BurgerKing(f *webfetch.Client) *PDFTable {
	return &PDFTable{
		Fetch: f, ID: "burgerking", Name: "バーガーキング", Origin: "https://www.burgerking.co.jp",
		License: "burgerking.co.jp terms of use (no reuse without permission)",
		Resolve: burgerKingPDF,
	}
}

func burgerKingPDF(ctx context.Context, t *PDFTable) ([]string, error) {
	msg := `{"header":{"result":true,"error_code":"","error_text":"","info_text":"","message_version":"","login_session_id":"","trcode":"BKJ0310"},"body":{"infoType":"02"}}`
	res, err := t.Fetch.PostForm(ctx, t.base()+"/burgerking/BKJ0310.json", url.Values{"message": {msg}})
	if err != nil {
		return nil, err
	}
	var body struct {
		Body struct {
			PDFURL string `json:"pdfUrl"`
		} `json:"body"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		return nil, err
	}
	if body.Body.PDFURL == "" {
		return nil, errors.New("burger king pdf url not found")
	}
	return []string{body.Body.PDFURL}, nil
}
