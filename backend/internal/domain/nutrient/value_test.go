package nutrient

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func TestForGrams(t *testing.T) {
	per100 := Profile{
		Energy:     Measured(d("343")),
		Protein:    Estimated(d("11.3")),
		VitaminB12: Missing(),
		VitaminD:   Trace(),
	}

	got := per100.ForGrams(d("150"))

	if !got.Get(Energy).Amount.Equal(d("514.5")) {
		t.Errorf("energy = %s", got.Get(Energy).Amount)
	}
	if got.Get(Protein).Status != StatusEstimated {
		t.Errorf("protein status = %s", got.Get(Protein).Status)
	}
	if got.Get(VitaminB12).Known() {
		t.Error("missing value became known")
	}
	if got.Get(VitaminD).Status != StatusTrace || !got.Get(VitaminD).Amount.IsZero() {
		t.Errorf("trace = %+v", got.Get(VitaminD))
	}
	if got.Get(Iron).Known() {
		t.Error("absent code must be missing")
	}
}

func TestSum(t *testing.T) {
	tests := []struct {
		name         string
		parts        []Part
		code         Code
		wantAmount   string
		wantStatus   Status
		wantCoverage string
	}{
		{
			name:         "no parts",
			parts:        nil,
			code:         Energy,
			wantAmount:   "0",
			wantStatus:   StatusCalculated,
			wantCoverage: "1",
		},
		{
			name: "all known",
			parts: []Part{
				{Grams: d("100"), Values: Profile{Energy: Measured(d("200"))}},
				{Grams: d("50"), Values: Profile{Energy: Estimated(d("80"))}},
			},
			code:         Energy,
			wantAmount:   "280",
			wantStatus:   StatusCalculated,
			wantCoverage: "1",
		},
		{
			name: "partially missing",
			parts: []Part{
				{Grams: d("300"), Values: Profile{VitaminB12: Measured(d("1.5"))}},
				{Grams: d("100"), Values: Profile{VitaminB12: Missing()}},
			},
			code:         VitaminB12,
			wantAmount:   "1.5",
			wantStatus:   StatusCalculated,
			wantCoverage: "0.75",
		},
		{
			name: "all missing stays missing",
			parts: []Part{
				{Grams: d("100"), Values: Profile{}},
				{Grams: d("100"), Values: Profile{VitaminB12: Missing()}},
			},
			code:         VitaminB12,
			wantAmount:   "0",
			wantStatus:   StatusMissing,
			wantCoverage: "0",
		},
		{
			name: "trace counts as known zero",
			parts: []Part{
				{Grams: d("100"), Values: Profile{VitaminD: Trace()}},
			},
			code:         VitaminD,
			wantAmount:   "0",
			wantStatus:   StatusCalculated,
			wantCoverage: "1",
		},
		{
			name: "nested coverage is weighted",
			parts: []Part{
				{Grams: d("100"), Values: Profile{Iron: {Amount: d("1"), Status: StatusCalculated, Coverage: d("0.5")}}},
				{Grams: d("100"), Values: Profile{Iron: Measured(d("2"))}},
			},
			code:         Iron,
			wantAmount:   "3",
			wantStatus:   StatusCalculated,
			wantCoverage: "0.75",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sum(tt.parts).Get(tt.code)
			if !got.Amount.Equal(d(tt.wantAmount)) {
				t.Errorf("amount = %s, want %s", got.Amount, tt.wantAmount)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", got.Status, tt.wantStatus)
			}
			if !got.Coverage.Equal(d(tt.wantCoverage)) {
				t.Errorf("coverage = %s, want %s", got.Coverage, tt.wantCoverage)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	if err := (Profile{Energy: {Amount: d("-1"), Status: StatusMeasured, Coverage: d("1")}}).Validate(); err == nil {
		t.Error("negative amount accepted")
	}
	if err := (Profile{Energy: {Amount: d("1"), Status: "unknown", Coverage: d("1")}}).Validate(); err == nil {
		t.Error("invalid status accepted")
	}
}
