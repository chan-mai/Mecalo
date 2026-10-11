package target

import (
	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

type Level string

const (
	LevelDeficient Level = "deficient"
	LevelLow       Level = "low"
	LevelAdequate  Level = "adequate"
	LevelHigh      Level = "high"
	LevelExcessive Level = "excessive"
	// 未計測・充足率不足
	LevelUnknown Level = "unknown"
	// 基準値なし
	LevelNone Level = "none"
)

// 不足判定に必要な最低充足率
var MinCoverage = decimal.RequireFromString("0.8")

var (
	energyLowRatio  = decimal.RequireFromString("0.9")
	energyHighRatio = decimal.RequireFromString("1.1")
)

type Evaluation struct {
	Code  nutrient.Code
	Level Level
	Value nutrient.Value
	// 比較に使った下限・上限
	Lower *decimal.Decimal
	Upper *decimal.Decimal
}

// 過剰側は判明分のみで判定, 不足側は充足率不足なら判定なし
func Evaluate(t Target, v nutrient.Value) Evaluation {
	e := Evaluation{Code: t.Code, Value: v}
	e.Lower, e.Upper = t.bounds()

	if e.Lower == nil && e.Upper == nil && t.UL == nil {
		e.Level = LevelNone
		return e
	}
	if !v.Known() {
		e.Level = LevelUnknown
		return e
	}

	amount := v.Amount
	switch {
	case t.Custom == nil && t.UL != nil && amount.GreaterThan(*t.UL):
		e.Level = LevelExcessive
		return e
	case e.Upper != nil && amount.GreaterThan(*e.Upper):
		e.Level = LevelHigh
		return e
	}

	if v.Coverage.LessThan(MinCoverage) {
		e.Level = LevelUnknown
		return e
	}

	// 下限の目安がない栄養素(食塩相当量等)は不足判定なし
	switch {
	case e.Lower == nil:
		e.Level = LevelAdequate
	case t.Custom == nil && t.EAR != nil && amount.LessThan(*t.EAR):
		e.Level = LevelDeficient
	case amount.LessThan(*e.Lower):
		e.Level = LevelLow
	default:
		e.Level = LevelAdequate
	}
	return e
}

func (t Target) bounds() (lower, upper *decimal.Decimal) {
	if t.Custom != nil {
		return t.Custom.Min, t.Custom.Max
	}
	if t.Goal != nil {
		l := t.Goal.Mul(energyLowRatio)
		u := t.Goal.Mul(energyHighRatio)
		return &l, &u
	}
	lower = firstNonNil(t.RDA, t.AI, t.DGMin)
	upper = t.DGMax
	return lower, upper
}

func firstNonNil(values ...*decimal.Decimal) *decimal.Decimal {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
