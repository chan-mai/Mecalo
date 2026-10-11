package nutrient

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusMeasured  Status = "measured"
	StatusEstimated Status = "estimated"
	// 微量, 集計上は0
	StatusTrace Status = "trace"
	// 未計測, 0とは区別
	StatusMissing Status = "missing"
	// 複数の値からの合算
	StatusCalculated Status = "calculated"
)

func (s Status) Valid() bool {
	switch s {
	case StatusMeasured, StatusEstimated, StatusTrace, StatusMissing, StatusCalculated:
		return true
	}
	return false
}

var (
	one     = decimal.NewFromInt(1)
	hundred = decimal.NewFromInt(100)
)

type Value struct {
	Amount decimal.Decimal
	Status Status
	// 元の重量のうち値が判明している割合(0〜1)
	Coverage decimal.Decimal
}

func Measured(amount decimal.Decimal) Value {
	return Value{Amount: amount, Status: StatusMeasured, Coverage: one}
}

func Estimated(amount decimal.Decimal) Value {
	return Value{Amount: amount, Status: StatusEstimated, Coverage: one}
}

func Trace() Value {
	return Value{Amount: decimal.Zero, Status: StatusTrace, Coverage: one}
}

func Missing() Value {
	return Value{Amount: decimal.Zero, Status: StatusMissing, Coverage: decimal.Zero}
}

func (v Value) Known() bool {
	return v.Status != StatusMissing
}

func (v Value) Validate() error {
	if !v.Status.Valid() {
		return fmt.Errorf("invalid nutrient status %q", v.Status)
	}
	if v.Amount.IsNegative() {
		return fmt.Errorf("negative nutrient amount %s", v.Amount)
	}
	if v.Coverage.IsNegative() || v.Coverage.GreaterThan(one) {
		return fmt.Errorf("coverage out of range %s", v.Coverage)
	}
	return nil
}

// 栄養素ごとの値, キーなしは未計測
type Profile map[Code]Value

func (p Profile) Get(code Code) Value {
	if v, ok := p[code]; ok {
		return v
	}
	return Missing()
}

func (p Profile) Validate() error {
	for code, v := range p {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("%s: %w", code, err)
		}
	}
	return nil
}

// 量のみ倍率適用, 状態と充足率は維持
func (p Profile) Scale(factor decimal.Decimal) Profile {
	out := make(Profile, len(p))
	for code, v := range p {
		v.Amount = v.Amount.Mul(factor)
		out[code] = v
	}
	return out
}

// 100gあたりの値から指定グラム分の値
func (p Profile) ForGrams(grams decimal.Decimal) Profile {
	return p.Scale(grams.Div(hundred))
}

// 合算対象の1要素
type Part struct {
	// 充足率の重み付けに使う重量(g)
	Grams  decimal.Decimal
	Values Profile
}

// 既知の値のみ加算し、栄養素ごとの充足率を重量比で算出
func Sum(parts []Part) Profile {
	codes := map[Code]struct{}{}
	for _, d := range catalog {
		codes[d.Code] = struct{}{}
	}
	for _, part := range parts {
		for code := range part.Values {
			codes[code] = struct{}{}
		}
	}

	totalGrams := decimal.Zero
	for _, part := range parts {
		totalGrams = totalGrams.Add(part.Grams)
	}

	out := make(Profile, len(codes))
	for code := range codes {
		amount := decimal.Zero
		knownGrams := decimal.Zero
		anyKnown := false
		for _, part := range parts {
			v := part.Values.Get(code)
			if !v.Known() {
				continue
			}
			anyKnown = true
			amount = amount.Add(v.Amount)
			knownGrams = knownGrams.Add(part.Grams.Mul(v.Coverage))
		}

		switch {
		case len(parts) > 0 && !anyKnown:
			out[code] = Missing()
		case totalGrams.IsZero():
			out[code] = Value{Amount: amount, Status: StatusCalculated, Coverage: one}
		default:
			out[code] = Value{Amount: amount, Status: StatusCalculated, Coverage: knownGrams.Div(totalGrams)}
		}
	}
	return out
}
