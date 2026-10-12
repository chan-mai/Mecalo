package catalog

import (
	"regexp"

	"github.com/shopspring/decimal"
)

var (
	headingUnit = regexp.MustCompile(`^(.*?)\s*\(?\s*(kcal|g|mg|μg)\s*\)?\s*$`)
	valueUnit   = regexp.MustCompile(`^(.*?)\s*(kcal|mg|μg|g)?$`)
	weightOnly  = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(g|ml)$`)
)

// 「熱量(kcal)」等の見出しを成分名と単位へ分割
func SplitHeading(s string) (string, string) {
	s = Normalize(s)
	if m := headingUnit.FindStringSubmatch(s); m != nil {
		return m[1], m[2]
	}
	return s, ""
}

// 「21kcal」を値と単位へ分割
func SplitValue(s string) (string, string) {
	m := valueUnit.FindStringSubmatch(Normalize(s))
	return m[1], m[2]
}

// 「80g」等の重量のみの表記, mlはgと同じ扱い
func Weight(s string) (decimal.Decimal, bool) {
	m := weightOnly.FindStringSubmatch(Normalize(s))
	if m == nil {
		return decimal.Zero, false
	}
	g, err := decimal.NewFromString(m[1])
	return g, err == nil
}
