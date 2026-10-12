package catalog

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

type pdfColumn struct {
	center float64
	name   string
	unit   string
}

var (
	pdfNumber = regexp.MustCompile(`^(?:\d[\d,]*(?:\.\d+)?|-|ー|Tr|tr)$`)
	pdfUnit   = regexp.MustCompile(`(?i)\(?(kcal|mg|μg|g)\)?`)
	// アレルゲン表の記号
	pdfSymbols = regexp.MustCompile(`[○●◯◎△▲×✕□■※]`)
)

// 見出しの栄養成分の列へ, 各行の数値を最も近い列で割り当て
func PDFRows(pages []PDFPage) []Row {
	var out []Row
	for _, page := range pages {
		lines := page.Lines()
		var (
			cols []pdfColumn
			// 直前のデータ行以降の見出し候補
			pending [][]PDFWord
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

func isDataLine(line []PDFWord) bool {
	n := 0
	for _, w := range line {
		if pdfNumber.MatchString(Normalize(w.Text)) && w.Text != "-" && w.Text != "ー" {
			n++
		}
	}
	return n >= 2
}

// 横書きの見出し行, なければ縦書きの見出しから列を作成
func headerOf(lines [][]PDFWord) []pdfColumn {
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
func verticalHeader(lines [][]PDFWord) []pdfColumn {
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
		text := Normalize(s.text.String())
		unit := ""
		if m := pdfUnit.FindStringSubmatch(text); m != nil {
			unit = strings.ToLower(m[1])
			text = strings.Replace(text, m[0], "", 1)
		}
		if IsNutrientName(text) {
			cols = append(cols, pdfColumn{center: s.center, name: text, unit: unit})
		}
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].center < cols[j].center })
	return cols
}

var sizeOnly = regexp.MustCompile(`^(?:S|M|L|LL|R|ショート|トール|グランデ|ベンティ|小|中|大|並|ミニ)$`)

// 縦結合された商品名のサイズ行に直前の商品名を補完
func sizeRows(rows []Row) []Row {
	base := ""
	for i, r := range rows {
		fields := strings.Fields(r.Name)
		switch {
		case len(fields) == 1 && sizeOnly.MatchString(fields[0]) && base != "":
			rows[i].Name = base + " " + fields[0]
		case len(fields) > 1 && sizeOnly.MatchString(fields[len(fields)-1]):
			base = strings.Join(fields[:len(fields)-1], " ")
		default:
			base = r.Name
		}
	}
	return rows
}

func center(w PDFWord) float64 {
	return w.X + w.W/2
}

func pdfHeader(line []PDFWord, below [][]PDFWord) []pdfColumn {
	var cols []pdfColumn
	for _, w := range line {
		name := Normalize(w.Text)
		base, unit := SplitHeading(name)
		if !IsNutrientName(base) {
			continue
		}
		c := pdfColumn{center: center(w), name: base, unit: unit}
		// 単位が次の行にある見出し
		if c.unit == "" {
			for _, l := range below {
				for _, u := range l {
					if m := pdfUnit.FindStringSubmatch(Normalize(u.Text)); m != nil && math.Abs(center(u)-c.center) < w.Size*2 {
						c.unit = strings.ToLower(m[1])
					}
				}
			}
		}
		cols = append(cols, c)
	}
	return cols
}

func pdfRowOf(line []PDFWord, cols []pdfColumn) (Row, bool) {
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
		values Values
		used   = map[int]bool{}
	)
	for _, w := range line {
		text := Normalize(w.Text)
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
		return Row{}, false
	}
	return Row{Name: n, Values: values}, true
}
