package food

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// 空白区切りの各語の検索キー
func SearchTerms(s string) []string {
	var out []string
	for _, w := range strings.Fields(norm.NFKC.String(s)) {
		if k := SearchKey(w); k != "" {
			out = append(out, k)
		}
	}
	return out
}

// 表記揺れ吸収用の検索キー, 語の区切りは半角空白1つ
func SearchKey(s string) string {
	s = strings.Join(strings.Fields(norm.NFKC.String(s)), " ")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'ァ' && r <= 'ヶ':
			b.WriteRune(r - ('ァ' - 'ぁ'))
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
