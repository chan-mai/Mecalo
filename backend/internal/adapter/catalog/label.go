// 公式サイト等の栄養成分表示の解析と一覧同期の共通部品
package catalog

import (
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

// 表示上の成分名
var labelNames = []struct {
	names []string
	code  nutrient.Code
}{
	{[]string{"エネルギー", "熱量", "カロリー"}, nutrient.Energy},
	{[]string{"たんぱく質", "タンパク質", "蛋白質", "たん白質"}, nutrient.Protein},
	{[]string{"脂質"}, nutrient.Fat},
	{[]string{"炭水化物"}, nutrient.Carbohydrate},
	{[]string{"食物繊維"}, nutrient.Fiber},
	{[]string{"食塩相当量", "食塩"}, nutrient.SaltEquivalent},
	{[]string{"ナトリウム"}, nutrient.Sodium},
	{[]string{"コレステロール"}, nutrient.Cholesterol},
	{[]string{"カリウム"}, nutrient.Potassium},
	{[]string{"カルシウム"}, nutrient.Calcium},
	{[]string{"マグネシウム"}, nutrient.Magnesium},
	{[]string{"リン"}, nutrient.Phosphorus},
	{[]string{"鉄"}, nutrient.Iron},
	{[]string{"亜鉛"}, nutrient.Zinc},
	{[]string{"ビタミンA"}, nutrient.VitaminA},
	{[]string{"ビタミンD"}, nutrient.VitaminD},
	{[]string{"ビタミンE"}, nutrient.VitaminE},
	{[]string{"ビタミンK"}, nutrient.VitaminK},
	{[]string{"ビタミンB1"}, nutrient.VitaminB1},
	{[]string{"ビタミンB2"}, nutrient.VitaminB2},
	{[]string{"ナイアシン"}, nutrient.Niacin},
	{[]string{"ビタミンB6"}, nutrient.VitaminB6},
	{[]string{"ビタミンB12"}, nutrient.VitaminB12},
	{[]string{"葉酸"}, nutrient.Folate},
	{[]string{"ビタミンC"}, nutrient.VitaminC},
}

// 炭水化物の内訳, 炭水化物未記載時の算出用
const sugarName = "糖質"

var unitFactor = map[string]map[nutrient.Unit]decimal.Decimal{
	"kcal": {nutrient.UnitKcal: decimal.NewFromInt(1)},
	"g": {
		nutrient.UnitGram:      decimal.NewFromInt(1),
		nutrient.UnitMilligram: decimal.NewFromInt(1_000),
		nutrient.UnitMicrogram: decimal.NewFromInt(1_000_000),
	},
	"mg": {
		nutrient.UnitGram:      decimal.New(1, -3),
		nutrient.UnitMilligram: decimal.NewFromInt(1),
		nutrient.UnitMicrogram: decimal.NewFromInt(1_000),
	},
	"μg": {
		nutrient.UnitGram:      decimal.New(1, -6),
		nutrient.UnitMilligram: decimal.New(1, -3),
		nutrient.UnitMicrogram: decimal.NewFromInt(1),
	},
}

// 全角・半角, 康熙部首等の表記揺れの統一
func Normalize(s string) string {
	s = norm.NFKC.String(s)
	s = strings.NewReplacer("µ", "μ", "ug", "μg", "ｇ", "g", " ", " ", "，", ",", "：", ":", "®", "", "™", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// 成分名と値の組, 単位はkcal・g・mg・μgのいずれか
type Values struct {
	byCode map[nutrient.Code]value
	sugar  *value
}

type value struct {
	amount decimal.Decimal
	unit   string
	// 範囲表示等の推定値
	estimated bool
}

// 成分名で設定, 未知の成分名はfalse
func (v *Values) Set(name, amount, unit string) bool {
	name = Normalize(name)
	// 空は成分の標準の単位
	unit = strings.ToLower(Normalize(unit))
	if strings.Contains(name, "kcal") {
		unit = "kcal"
	}
	val, ok := parseAmount(amount)
	if !ok {
		return false
	}
	val.unit = unit
	if strings.HasPrefix(name, sugarName) {
		v.sugar = &val
		return true
	}
	code, ok := codeOf(name)
	if !ok {
		return false
	}
	if v.byCode == nil {
		v.byCode = map[nutrient.Code]value{}
	}
	if _, dup := v.byCode[code]; !dup {
		v.byCode[code] = val
	}
	return true
}

// 数値指定での設定
func (v *Values) SetNumber(code nutrient.Code, amount decimal.Decimal, unit string) {
	if v.byCode == nil {
		v.byCode = map[nutrient.Code]value{}
	}
	v.byCode[code] = value{amount: amount, unit: unit}
}

func (v *Values) SetSugar(amount decimal.Decimal) {
	v.sugar = &value{amount: amount, unit: "g"}
}

func (v Values) Len() int {
	return len(v.byCode)
}

// 未記載の成分はキーなし(未計測)
func (v Values) Profile() nutrient.Profile {
	p := nutrient.Profile{}
	for code, val := range v.byCode {
		def, ok := nutrient.Lookup(code)
		if !ok {
			continue
		}
		f, ok := unitFactor[val.unit][def.Unit]
		if val.unit == "" {
			f, ok = decimal.NewFromInt(1), true
		}
		if !ok {
			continue
		}
		amount := val.amount.Mul(f)
		if val.estimated {
			p[code] = nutrient.Estimated(amount)
		} else {
			p[code] = nutrient.Measured(amount)
		}
	}
	// 炭水化物 = 糖質 + 食物繊維
	if _, ok := p[nutrient.Carbohydrate]; !ok && v.sugar != nil {
		if fib, ok := p[nutrient.Fiber]; ok {
			p[nutrient.Carbohydrate] = nutrient.Estimated(v.sugar.amount.Add(fib.Amount))
		}
	}
	// 食塩相当量 = ナトリウム(mg) × 2.54 / 1000
	if _, ok := p[nutrient.SaltEquivalent]; !ok {
		if na, ok := p[nutrient.Sodium]; ok {
			p[nutrient.SaltEquivalent] = nutrient.Estimated(na.Amount.Mul(decimal.RequireFromString("2.54")).Div(decimal.NewFromInt(1000)).Round(2))
		}
	}
	return p
}

func codeOf(name string) (nutrient.Code, bool) {
	for _, l := range labelNames {
		for _, n := range l.names {
			if strings.HasPrefix(name, n) {
				// ビタミンB1とB12等の前方一致の衝突回避
				if rest := strings.TrimPrefix(name, n); rest != "" && rest[0] >= '0' && rest[0] <= '9' {
					continue
				}
				return l.code, true
			}
		}
	}
	return "", false
}

var rangePattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[~〜-]\s*(\d+(?:\.\d+)?)$`)

// 「1,005」「0~1」「Tr」「-」に対応, 未分析はfalse
func parseAmount(s string) (value, bool) {
	s = strings.ReplaceAll(Normalize(s), ",", "")
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "-", "ー", "―", "未分析", "n/a":
		return value{}, false
	case "tr", "微量":
		return value{amount: decimal.Zero, estimated: true}, true
	}
	if m := rangePattern.FindStringSubmatch(s); m != nil {
		lo, _ := decimal.NewFromString(m[1])
		hi, _ := decimal.NewFromString(m[2])
		return value{amount: lo.Add(hi).Div(decimal.NewFromInt(2)), estimated: true}, true
	}
	s = strings.TrimPrefix(s, "約")
	d, err := decimal.NewFromString(s)
	if err != nil || d.IsNegative() {
		return value{}, false
	}
	return value{amount: d}, true
}

var (
	labelItem = regexp.MustCompile(`(?i)([^\s\d:()（）,、]+?)\s*:?\s*(\d[\d,]*(?:\.\d+)?(?:\s*[~〜]\s*\d+(?:\.\d+)?)?|Tr|-)\s*\(?(kcal|mg|μg|g)?\)?`)
	basisText = regexp.MustCompile(`(?:^|\s)([^\s]*?)(?:\((\d+(?:\.\d+)?)\s*(g|ml)\))?\s*(?:当たり|あたり)`)
	weightIn  = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(g|ml)`)
)

// 「1袋(240g)当たり エネルギー322kcal …」形式の解析, 基準量の記載なしはfound=false
func ParseLabel(text string) (basis food.Basis, found bool, v Values) {
	text = Normalize(text)
	if m := basisText.FindStringSubmatch(text); m != nil {
		basis, found = BasisOf(m[1], m[2]), true
		text = strings.Replace(text, m[0], " ", 1)
	}
	for _, m := range labelItem.FindAllStringSubmatch(text, -1) {
		v.Set(m[1], m[2], m[3])
	}
	return basis, found, v
}

// 基準量の表記と重量から基準量を生成, 100gは単位なし
func BasisOf(label, grams string) food.Basis {
	label = strings.TrimSpace(Normalize(label))
	if grams == "" {
		if m := weightIn.FindStringSubmatch(label); m != nil && m[0] == label {
			grams = m[1]
		}
	}
	g, _ := decimal.NewFromString(grams)
	if label == "100g" || label == "100ml" {
		return food.Basis{}
	}
	if m := weightIn.FindStringSubmatch(label); m != nil && m[0] == label {
		// 「50g当たり」等の重量基準
		return food.Basis{Unit: label, Grams: g}
	}
	if label == "" {
		label = "1食"
	}
	return food.Basis{Unit: label, Grams: g}
}

// 栄養成分の見出しとして扱える名前
func IsNutrientName(name string) bool {
	name = Normalize(name)
	if strings.HasPrefix(name, sugarName) {
		return true
	}
	_, ok := codeOf(name)
	return ok
}
