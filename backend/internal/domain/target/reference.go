// 食事摂取基準に基づく目標量の算出と評価
package target

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

// 年齢区分, Maxは含まず, 0は上限なし
type AgeBand struct {
	Label string
	Min   int
	Max   int
}

func (b AgeBand) Contains(age int) bool {
	return age >= b.Min && (b.Max == 0 || age < b.Max)
}

// 基準値の性別区分
type Group string

const (
	GroupMale               Group = "male"
	GroupFemale             Group = "female"
	GroupFemaleMenstruating Group = "female_menstruating"
)

// 未設定はnil
type Indicators struct {
	EAR   *decimal.Decimal
	RDA   *decimal.Decimal
	AI    *decimal.Decimal
	UL    *decimal.Decimal
	DGMin *decimal.Decimal
	DGMax *decimal.Decimal
	// エネルギー比(%)の目標量
	DGEnergyPctMin *decimal.Decimal
	DGEnergyPctMax *decimal.Decimal
}

type NutrientReference struct {
	Source string
	Note   string
	// Group → 年齢区分ラベル → 指標
	Values map[Group]map[string]Indicators
}

type Reference struct {
	Edition string
	Title   string
	URL     string
	Bands   []AgeBand
	// 基礎代謝基準値(kcal/kg体重/日)
	BMRPerKg map[body.Sex]map[string]decimal.Decimal
	// 身体活動レベル, 区分なしはキーなし
	PAL       map[string]map[body.ActivityLevel]decimal.Decimal
	Nutrients map[nutrient.Code]NutrientReference
}

func (r Reference) BandFor(age int) (AgeBand, error) {
	for _, b := range r.Bands {
		if b.Contains(age) {
			return b, nil
		}
	}
	return AgeBand{}, fmt.Errorf("no reference band for age %d", age)
}

func (r Reference) indicators(code nutrient.Code, groups []Group, band string) (Indicators, bool) {
	ref, ok := r.Nutrients[code]
	if !ok {
		return Indicators{}, false
	}
	for _, g := range groups {
		if byBand, ok := ref.Values[g]; ok {
			if ind, ok := byBand[band]; ok {
				return ind, true
			}
		}
	}
	return Indicators{}, false
}
