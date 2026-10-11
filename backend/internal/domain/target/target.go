package target

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

// kcal/g
var atwater = map[nutrient.Code]decimal.Decimal{
	nutrient.Protein:      decimal.NewFromInt(4),
	nutrient.Fat:          decimal.NewFromInt(9),
	nutrient.Carbohydrate: decimal.NewFromInt(4),
}

type Person struct {
	Sex          body.Sex
	Age          int
	WeightKg     decimal.Decimal
	Activity     body.ActivityLevel
	Menstruating bool
}

// 利用者が設定した目標範囲
type Override struct {
	Min *decimal.Decimal
	Max *decimal.Decimal
}

type Target struct {
	Code nutrient.Code
	Indicators
	// エネルギーの目標値
	Goal   *decimal.Decimal
	Custom *Override
}

type Targets struct {
	Edition    string
	Band       string
	EnergyKcal decimal.Decimal
	ByCode     map[nutrient.Code]Target
}

// 基礎代謝基準値 × 体重 × 身体活動レベル
func EstimatedEnergy(ref Reference, p Person) (decimal.Decimal, error) {
	if !p.WeightKg.IsPositive() {
		return decimal.Zero, errors.New("weight is required")
	}
	band, err := ref.BandFor(p.Age)
	if err != nil {
		return decimal.Zero, err
	}
	bmr, ok := ref.BMRPerKg[p.Sex][band.Label]
	if !ok {
		return decimal.Zero, fmt.Errorf("no basal metabolic rate for %s %s", p.Sex, band.Label)
	}
	pal, ok := ref.PAL[band.Label][p.Activity]
	if !ok {
		pal, ok = ref.PAL[band.Label][body.ActivityModerate]
		if !ok {
			return decimal.Zero, fmt.Errorf("no physical activity level for %s", band.Label)
		}
	}
	return bmr.Mul(p.WeightKg).Mul(pal), nil
}

func Build(ref Reference, p Person, energyAdjust decimal.Decimal, overrides map[nutrient.Code]Override) (Targets, error) {
	band, err := ref.BandFor(p.Age)
	if err != nil {
		return Targets{}, err
	}
	eer, err := EstimatedEnergy(ref, p)
	if err != nil {
		return Targets{}, err
	}
	energy := eer.Add(energyAdjust)
	if !energy.IsPositive() {
		return Targets{}, errors.New("energy target must be positive")
	}

	groups := []Group{GroupMale}
	if p.Sex == body.SexFemale {
		groups = []Group{GroupFemale}
		if p.Menstruating {
			groups = []Group{GroupFemaleMenstruating, GroupFemale}
		}
	}

	out := Targets{
		Edition:    ref.Edition,
		Band:       band.Label,
		EnergyKcal: energy,
		ByCode:     map[nutrient.Code]Target{},
	}
	out.ByCode[nutrient.Energy] = Target{Code: nutrient.Energy, Goal: &energy}

	for _, def := range nutrient.Catalog() {
		if def.Code == nutrient.Energy {
			continue
		}
		ind, ok := ref.indicators(def.Code, groups, band.Label)
		if !ok {
			continue
		}
		out.ByCode[def.Code] = Target{Code: def.Code, Indicators: withEnergyRatio(def.Code, ind, energy)}
	}

	for code, o := range overrides {
		o := o
		t := out.ByCode[code]
		t.Code = code
		t.Custom = &o
		out.ByCode[code] = t
	}
	return out, nil
}

// エネルギー比の目標量をグラムへ換算
func withEnergyRatio(code nutrient.Code, ind Indicators, energy decimal.Decimal) Indicators {
	kcalPerGram, ok := atwater[code]
	if !ok {
		return ind
	}
	toGrams := func(pct *decimal.Decimal) *decimal.Decimal {
		if pct == nil {
			return nil
		}
		g := energy.Mul(*pct).Div(decimal.NewFromInt(100)).Div(kcalPerGram)
		return &g
	}
	if ind.DGMin == nil {
		ind.DGMin = toGrams(ind.DGEnergyPctMin)
	}
	if ind.DGMax == nil {
		ind.DGMax = toGrams(ind.DGEnergyPctMax)
	}
	return ind
}
