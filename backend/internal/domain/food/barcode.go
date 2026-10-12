package food

import "strings"

// JAN・UPC・GTINの照合用表記, 不正な桁数はfalse
func NormalizeBarcode(code string) (string, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	switch len(code) {
	case 8:
		return code, true
	case 12:
		return "0" + code, true
	case 13:
		// 先頭5桁の0埋めは短縮JAN
		if strings.HasPrefix(code, "00000") {
			return code[5:], true
		}
		return code, true
	case 14:
		if code[0] != '0' {
			// 梱包単位のGTIN-14は単品と別商品
			return code, true
		}
		return NormalizeBarcode(code[1:])
	}
	return "", false
}

// GTIN共通のチェックディジット検証
func ValidBarcode(code string) bool {
	n, ok := NormalizeBarcode(code)
	if !ok || n != code {
		return false
	}
	sum := 0
	weight := 3
	for i := len(code) - 2; i >= 0; i-- {
		sum += int(code[i]-'0') * weight
		weight = 4 - weight
	}
	return (10-sum%10)%10 == int(code[len(code)-1]-'0')
}
