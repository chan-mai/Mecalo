// 日本食品標準成分表(八訂)本表Excelの読込
package mext

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

const (
	// 増補2023年 本表
	DefaultURL = "https://www.mext.go.jp/content/20260327-mxt_kagsei-mext-000029402_02.xlsx"
	DatasetID  = food.DatasetID("mext")
	sheetName  = "表全体"
	// 成分識別子の見出し
	tagHeader    = "成分識別子"
	remarkHeader = "備考"
)

var (
	foodNumberPattern = regexp.MustCompile(`^\d{5}$`)
	aliasPattern      = regexp.MustCompile(`^\s*※?\s*別名\s*[:\x{FF1A}]\s*(.+)$`)
	aliasSeparator    = regexp.MustCompile(`[、,\x{FF0C}]`)
	updatedPattern    = regexp.MustCompile(`更新日[:\x{FF1A}]\s*(\d{4})年(\d{1,2})月(\d{1,2})日`)
)

type Table struct {
	// Excel記載の更新日(YYYY-MM-DD)
	UpdatedAt string
	Foods     []app.SystemFood
}

func Parse(r io.Reader) (Table, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return Table{}, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return Table{}, fmt.Errorf("read sheet %q: %w", sheetName, err)
	}

	var t Table
	columns := map[int]nutrient.Code{}
	nameCol := -1
	remarkCol := -1
	for _, row := range rows {
		if remarkCol < 0 {
			for i, cell := range row {
				if strings.Join(strings.Fields(strings.ReplaceAll(cell, "　", " ")), "") == remarkHeader {
					remarkCol = i
				}
			}
		}
		if t.UpdatedAt == "" {
			if m := updatedPattern.FindStringSubmatch(strings.Join(row, " ")); m != nil {
				t.UpdatedAt = fmt.Sprintf("%s-%02s-%02s", m[1], m[2], m[3])
			}
		}
		if nameCol < 0 {
			for i, cell := range row {
				if strings.TrimSpace(cell) == tagHeader {
					nameCol = i
				}
			}
			if nameCol >= 0 {
				for i, cell := range row {
					if def, ok := nutrient.ByInfoods(strings.TrimSpace(cell)); ok {
						columns[i] = def.Code
					}
				}
			}
			continue
		}
		if len(row) <= nameCol || len(row) < 2 || !foodNumberPattern.MatchString(strings.TrimSpace(row[1])) {
			continue
		}
		number := strings.TrimSpace(row[1])
		sf := app.SystemFood{
			ID:        food.ID("mext-" + number),
			Kind:      food.KindStandard,
			Name:      normalizeName(row[nameCol]),
			Reference: number,
			Nutrients: nutrient.Profile{},
		}
		if remarkCol >= 0 && remarkCol < len(row) {
			sf.Aliases = ParseAliases(row[remarkCol])
		}
		for col, code := range columns {
			if col >= len(row) {
				continue
			}
			v, err := ParseValue(row[col])
			if err != nil {
				return Table{}, fmt.Errorf("food %s %s: %w", number, code, err)
			}
			if v.Known() {
				sf.Nutrients[code] = v
			}
		}
		t.Foods = append(t.Foods, sf)
	}
	if nameCol < 0 {
		return Table{}, errors.New("header row with component identifiers not found")
	}
	if len(columns) == 0 || len(t.Foods) == 0 {
		return Table{}, errors.New("no foods found")
	}
	return t, nil
}

// "-" は未測定, "Tr" は微量, 括弧付きは推定値
func ParseValue(cell string) (nutrient.Value, error) {
	s := strings.TrimSpace(cell)
	// † は分析法の注記
	s = strings.NewReplacer(",", "", "　", "", " ", "", "†", "").Replace(s)
	switch s {
	case "", "-", "－", "*":
		return nutrient.Missing(), nil
	case "Tr":
		return nutrient.Trace(), nil
	}
	estimated := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		estimated = true
		s = strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	}
	if s == "Tr" {
		return nutrient.Trace(), nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nutrient.Value{}, fmt.Errorf("unexpected value %q", cell)
	}
	if estimated {
		return nutrient.Estimated(d), nil
	}
	return nutrient.Measured(d), nil
}

// 備考欄の「別名:」行から別名を抽出
func ParseAliases(remark string) []string {
	var out []string
	for _, line := range strings.Split(remark, "\n") {
		m := aliasPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, a := range aliasSeparator.Split(m[1], -1) {
			if a = strings.TrimSpace(a); a != "" {
				out = append(out, a)
			}
		}
	}
	return out
}

func normalizeName(s string) string {
	s = strings.ReplaceAll(s, "　", " ")
	return strings.Join(strings.Fields(s), " ")
}

func Dataset(updatedAt string) food.Dataset {
	return food.Dataset{
		ID:          DatasetID,
		Kind:        food.DatasetMEXT,
		Name:        "日本食品標準成分表(八訂)増補2023年",
		Version:     updatedAt,
		URL:         "https://www.mext.go.jp/a_menu/syokuhinseibun/mext_00001.html",
		Attribution: "文部科学省「日本食品標準成分表(八訂)増補2023年」",
	}
}
