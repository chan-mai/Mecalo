package catalog

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

// PDF内の1語, 座標は上が大きい
type PDFWord struct {
	X, Y float64
	// 横幅
	W float64
	// 文字の高さ
	Size float64
	Text string
}

type PDFPage struct {
	Words []PDFWord
}

var (
	pdfOnce sync.Once
	pdfPool pdfium.Pool
	pdfErr  error
)

// 同時に解析するPDFは1つ, 初回利用時にPDFiumを起動
func pdfInstance() (pdfium.Pdfium, error) {
	pdfOnce.Do(func() {
		pdfPool, pdfErr = webassembly.Init(webassembly.Config{MinIdle: 0, MaxIdle: 1, MaxTotal: 1})
	})
	if pdfErr != nil {
		return nil, pdfErr
	}
	return pdfPool.GetInstance(5 * time.Minute)
}

// 埋め込み文字の語単位での取得, OCRなし
func ReadPDF(body []byte) ([]PDFPage, error) {
	inst, err := pdfInstance()
	if err != nil {
		return nil, fmt.Errorf("start pdfium: %w", err)
	}
	defer inst.Close()
	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &body})
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer func() { _, _ = inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document}) }()
	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return nil, err
	}
	pages := make([]PDFPage, 0, count.PageCount)
	for i := range count.PageCount {
		res, err := inst.GetPageTextStructured(&requests.GetPageTextStructured{
			Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}},
			Mode: requests.GetPageTextStructuredModeRects,
		})
		if err != nil {
			return nil, fmt.Errorf("read pdf page %d: %w", i+1, err)
		}
		var page PDFPage
		for _, r := range res.Rects {
			page.Words = append(page.Words, splitRect(r.Text, r.PointPosition.Left, r.PointPosition.Right, r.PointPosition.Top, r.PointPosition.Bottom)...)
		}
		pages = append(pages, page)
	}
	if len(pages) == 0 {
		return nil, errors.New("pdf has no pages")
	}
	return pages, nil
}

// 文字領域を空白で語へ分割, 各語のxは文字数の比率で推定
func splitRect(text string, left, right, top, bottom float64) []PDFWord {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	per := (right - left) / float64(len(runes))
	y, size := (top+bottom)/2, math.Abs(top-bottom)
	var out []PDFWord
	start := -1
	for i := 0; i <= len(runes); i++ {
		space := i == len(runes) || runes[i] == ' ' || runes[i] == '\u3000' || runes[i] == '\t'
		switch {
		case space && start >= 0:
			out = append(out, PDFWord{X: left + per*float64(start), W: per * float64(i-start), Y: y, Size: size, Text: string(runes[start:i])})
			start = -1
		case !space && start < 0:
			start = i
		}
	}
	return out
}

// y座標の近い語を1行にまとめ, 上の行から左から右の順
func (p PDFPage) Lines() [][]PDFWord {
	ws := append([]PDFWord(nil), p.Words...)
	sort.SliceStable(ws, func(i, j int) bool { return ws[i].Y > ws[j].Y })
	var lines [][]PDFWord
	for _, w := range ws {
		n := len(lines)
		if n > 0 && math.Abs(lines[n-1][0].Y-w.Y) <= math.Max(w.Size*0.5, 1) {
			lines[n-1] = append(lines[n-1], w)
			continue
		}
		lines = append(lines, []PDFWord{w})
	}
	for i, l := range lines {
		sort.Slice(l, func(i, j int) bool { return l[i].X < l[j].X })
		lines[i] = joinAdjacent(l)
	}
	return lines
}

// 1文字ずつ分かれた文字領域の連結
func joinAdjacent(line []PDFWord) []PDFWord {
	out := make([]PDFWord, 0, len(line))
	for _, w := range line {
		n := len(out)
		if n > 0 {
			prev := &out[n-1]
			if gap := w.X - (prev.X + prev.W); gap < math.Min(prev.Size, w.Size)*0.15 {
				prev.Text += w.Text
				prev.W = w.X + w.W - prev.X
				continue
			}
		}
		out = append(out, w)
	}
	return out
}
