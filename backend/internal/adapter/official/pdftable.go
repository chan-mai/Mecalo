package official

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
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
		pages, err := catalog.ReadPDF(body)
		if err != nil {
			return prev, err
		}
		for _, row := range pdfRows(pages) {
			ref := food.SearchKey(row.name)
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
				Name:      row.name,
				Brand:     t.Name,
				Reference: ref,
				Nutrients: row.values.Profile(),
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
	doc, err := catalog.ParseHTML(res.Body)
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

type pdfColumn struct {
	center float64
	name   string
	unit   string
}

type pdfRow struct {
	name   string
	values catalog.Values
}

var (
	pdfNumber = regexp.MustCompile(`^(?:\d[\d,]*(?:\.\d+)?|-|ー|Tr|tr)$`)
	pdfUnit   = regexp.MustCompile(`(?i)\(?(kcal|mg|μg|g)\)?`)
	// アレルゲン表の記号
	pdfSymbols = regexp.MustCompile(`[○●◯◎△▲×✕□■※]`)
)

// 見出しの栄養成分の列へ, 各行の数値を最も近い列で割り当て
func pdfRows(pages []catalog.PDFPage) []pdfRow {
	var out []pdfRow
	for _, page := range pages {
		lines := page.Lines()
		var (
			cols []pdfColumn
			// 直前のデータ行以降の見出し候補
			pending [][]catalog.PDFWord
		)
		for _, line := range lines {
			if !isDataLine(line) {
				pending = append(pending, line)
				continue
			}
			if len(pending) > 0 {
				if c := headerOf(pending); len(c) >= 2 {
					cols = c
				}
				pending = nil
			}
			if len(cols) == 0 {
				continue
			}
			if row, ok := pdfRowOf(line, cols); ok {
				out = append(out, row)
			}
		}
	}
	return sizeRows(out)
}

func isDataLine(line []catalog.PDFWord) bool {
	n := 0
	for _, w := range line {
		if pdfNumber.MatchString(catalog.Normalize(w.Text)) && w.Text != "-" && w.Text != "ー" {
			n++
		}
	}
	return n >= 2
}

// 横書きの見出し行, なければ縦書きの見出しから列を作成
func headerOf(lines [][]catalog.PDFWord) []pdfColumn {
	var best []pdfColumn
	for i, line := range lines {
		if c := pdfHeader(line, lines[i+1:]); len(c) > len(best) {
			best = c
		}
	}
	if v := verticalHeader(lines); len(v) > len(best) {
		best = v
	}
	return best
}

// 1文字ずつ縦に並ぶ見出しを横位置ごとに連結
func verticalHeader(lines [][]catalog.PDFWord) []pdfColumn {
	type stack struct {
		center float64
		size   float64
		text   strings.Builder
	}
	var stacks []*stack
	for _, line := range lines {
		for _, w := range line {
			if len([]rune(w.Text)) > 3 {
				continue
			}
			c := center(w)
			var s *stack
			for _, x := range stacks {
				if math.Abs(x.center-c) <= math.Max(w.Size, x.size)*0.6 {
					s = x
					break
				}
			}
			if s == nil {
				s = &stack{center: c, size: w.Size}
				stacks = append(stacks, s)
			}
			s.text.WriteString(w.Text)
		}
	}
	var cols []pdfColumn
	for _, s := range stacks {
		text := catalog.Normalize(s.text.String())
		unit := ""
		if m := pdfUnit.FindStringSubmatch(text); m != nil {
			unit = strings.ToLower(m[1])
			text = strings.Replace(text, m[0], "", 1)
		}
		if catalog.IsNutrientName(text) {
			cols = append(cols, pdfColumn{center: s.center, name: text, unit: unit})
		}
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].center < cols[j].center })
	return cols
}

var sizeOnly = regexp.MustCompile(`^(?:S|M|L|LL|R|ショート|トール|グランデ|ベンティ|小|中|大|並|ミニ)$`)

// 縦結合された商品名のサイズ行に直前の商品名を補完
func sizeRows(rows []pdfRow) []pdfRow {
	base := ""
	for i, r := range rows {
		fields := strings.Fields(r.name)
		switch {
		case len(fields) == 1 && sizeOnly.MatchString(fields[0]) && base != "":
			rows[i].name = base + " " + fields[0]
		case len(fields) > 1 && sizeOnly.MatchString(fields[len(fields)-1]):
			base = strings.Join(fields[:len(fields)-1], " ")
		default:
			base = r.name
		}
	}
	return rows
}

func center(w catalog.PDFWord) float64 {
	return w.X + w.W/2
}

func pdfHeader(line []catalog.PDFWord, below [][]catalog.PDFWord) []pdfColumn {
	var cols []pdfColumn
	for _, w := range line {
		name := catalog.Normalize(w.Text)
		base, unit := splitHeading(name)
		if !catalog.IsNutrientName(base) {
			continue
		}
		c := pdfColumn{center: center(w), name: base, unit: unit}
		// 単位が次の行にある見出し
		if c.unit == "" {
			for _, l := range below {
				for _, u := range l {
					if m := pdfUnit.FindStringSubmatch(catalog.Normalize(u.Text)); m != nil && math.Abs(center(u)-c.center) < w.Size*2 {
						c.unit = strings.ToLower(m[1])
					}
				}
			}
		}
		cols = append(cols, c)
	}
	return cols
}

func pdfRowOf(line []catalog.PDFWord, cols []pdfColumn) (pdfRow, bool) {
	spacing := math.Inf(1)
	for i := 1; i < len(cols); i++ {
		spacing = math.Min(spacing, math.Abs(cols[i].center-cols[i-1].center))
	}
	first := math.Inf(1)
	for _, c := range cols {
		first = math.Min(first, c.center)
	}
	var (
		name   []string
		values catalog.Values
		used   = map[int]bool{}
	)
	for _, w := range line {
		text := catalog.Normalize(w.Text)
		if !pdfNumber.MatchString(text) {
			if t := pdfSymbols.ReplaceAllString(text, ""); t != "" && center(w) < first-spacing/2 {
				name = append(name, t)
			}
			continue
		}
		best, dist := -1, spacing*0.6
		for i, c := range cols {
			if d := math.Abs(center(w) - c.center); d < dist && !used[i] {
				best, dist = i, d
			}
		}
		if best < 0 {
			continue
		}
		used[best] = true
		values.Set(cols[best].name, text, cols[best].unit)
	}
	n := strings.TrimSpace(strings.Join(name, " "))
	if n == "" || values.Len() == 0 {
		return pdfRow{}, false
	}
	return pdfRow{name: n, values: values}, true
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
