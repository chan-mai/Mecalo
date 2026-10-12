package catalog

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

// 文字列の並びから1行分の語, xは文字数×10
func pdfLine(y float64, cells ...string) []PDFWord {
	var out []PDFWord
	x := 0.0
	for _, c := range cells {
		if c != "" {
			w := float64(len([]rune(c))) * 5
			out = append(out, PDFWord{X: x, W: w, Y: y, Size: 10, Text: c})
		}
		x += 60
	}
	return out
}

func pdfPage(lines ...[]PDFWord) PDFPage {
	var p PDFPage
	for _, l := range lines {
		p.Words = append(p.Words, l...)
	}
	return p
}

func TestPDFRowsHorizontal(t *testing.T) {
	rows := PDFRows([]PDFPage{pdfPage(
		pdfLine(500, "", "エネルギー", "たんぱく質", "食塩相当量", "ナトリウム"),
		pdfLine(490, "", "(kcal)", "(g)", "(g)"),
		pdfLine(480, "テストバーガー", "750", "32.7", "3.7", "1,356", "●"),
		pdfLine(470, "テストポテト", "-", "2.9", "0.6", "200"),
	)})
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	p := rows[0].Values.Profile()
	if !p[nutrient.Energy].Amount.Equal(dec("750")) || !p[nutrient.Sodium].Amount.Equal(dec("1356")) {
		t.Errorf("burger = %v", p)
	}
	if rows[0].Name != "テストバーガー" {
		t.Errorf("name = %q", rows[0].Name)
	}
	if _, ok := rows[1].Values.Profile()[nutrient.Energy]; ok {
		t.Error("'-' must be missing")
	}
}

func TestPDFRowsVertical(t *testing.T) {
	rows := PDFRows([]PDFPage{pdfPage(
		pdfLine(530, "", "エ", "た", "食"),
		pdfLine(520, "", "ネ", "ん", "塩"),
		pdfLine(510, "", "ル", "ぱ", "相"),
		pdfLine(500, "メニュー名", "ギ", "く", "当"),
		pdfLine(490, "", "ー", "質", "量"),
		pdfLine(480, "テスト定食", "779", "26.5", "2.2"),
		pdfLine(470, "テストティー", "S", "100", "0.2"),
		pdfLine(460, "M", "", "150", "0.3"),
	)})
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	if v := rows[0].Values.Profile()[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("2.2")) {
		t.Errorf("salt = %v", v)
	}
	if rows[2].Name != "テストティー M" {
		t.Errorf("size row name = %q", rows[2].Name)
	}
}
