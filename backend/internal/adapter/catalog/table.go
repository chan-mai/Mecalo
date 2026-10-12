package catalog

import "strings"

// 栄養成分表の1行
type Row struct {
	Name string
	// 商品ページへのリンク, なければ空
	Href   string
	Values Values
}

// 見出しにエネルギー等を含む表の各行, 1列目は商品名
func TableRows(doc Node) []Row {
	var out []Row
	for _, table := range doc.FindAll("table", "") {
		var heads []string
		for _, tr := range table.FindAll("tr", "") {
			if ths := tr.FindAll("th", ""); len(ths) > 1 {
				heads = heads[:0]
				for _, th := range ths {
					heads = append(heads, th.Text())
				}
			}
		}
		joined := strings.Join(heads, " ")
		if !strings.Contains(joined, "エネルギー") && !strings.Contains(joined, "熱量") {
			continue
		}
		for _, tr := range table.FindAll("tr", "") {
			cells := tr.Cells()
			if len(cells) < 2 || cells[1].Data != "td" {
				continue
			}
			// 商品名の見出しが別の行にある表は見出しが1列少ない
			offset := len(cells) - len(heads)
			if offset < 0 || offset > 1 {
				continue
			}
			name := Normalize(cells[0].Text())
			var v Values
			for i := 1; i < len(cells); i++ {
				h := i - offset
				if h < 0 || h >= len(heads) {
					continue
				}
				n, unit := SplitHeading(heads[h])
				amount, valueUnit := SplitValue(cells[i].Text())
				if valueUnit != "" {
					unit = valueUnit
				}
				v.Set(n, amount, unit)
			}
			if name == "" || v.Len() == 0 {
				continue
			}
			out = append(out, Row{Name: name, Href: cells[0].Find("a", "").Attr("href"), Values: v})
		}
	}
	return out
}
