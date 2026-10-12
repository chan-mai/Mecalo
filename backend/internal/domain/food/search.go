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

// 類似度検索用の2文字の組, 記号と空白は除外, 重複なし
func Bigrams(key string) []string {
	var runes []rune
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			runes = append(runes, r)
		}
	}
	seen := map[string]bool{}
	var out []string
	for i := 0; i+1 < len(runes); i++ {
		g := string(runes[i : i+2])
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	return out
}

// 複数の検索キーの組の和集合
func BigramsOf(keys ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, k := range keys {
		for _, g := range Bigrams(k) {
			if !seen[g] {
				seen[g] = true
				out = append(out, g)
			}
		}
	}
	return out
}

// 検索語の組のうち候補が含む割合と, 組の集合の重なり(Dice係数)
func Similarity(shared, query, candidate int) (containment, dice float64) {
	if query == 0 || candidate == 0 {
		return 0, 0
	}
	return float64(shared) / float64(query), 2 * float64(shared) / float64(query+candidate)
}
