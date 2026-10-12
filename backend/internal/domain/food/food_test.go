package food

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func TestRevisionGrams(t *testing.T) {
	rev := Revision{
		Name:     "ごはん",
		Portions: []Portion{{Unit: "杯", Grams: d("150")}, {Unit: "人前", Grams: d("200")}},
	}

	tests := []struct {
		name    string
		q       Quantity
		want    string
		wantErr error
	}{
		{"gram", Quantity{d("120"), UnitGram}, "120", nil},
		{"portion", Quantity{d("1.5"), "杯"}, "225", nil},
		{"unknown unit", Quantity{d("1"), "個"}, "", ErrUnknownUnit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rev.Grams(tt.q)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(d(tt.want)) {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}

	if _, err := rev.Grams(Quantity{d("0"), UnitGram}); err == nil {
		t.Error("zero quantity accepted")
	}
}

func TestRevisionValidate(t *testing.T) {
	if err := (Revision{Name: " "}).Validate(); err == nil {
		t.Error("blank name accepted")
	}
	dup := Revision{Name: "a", Portions: []Portion{{"個", d("1")}, {"個", d("2")}}}
	if err := dup.Validate(); err == nil {
		t.Error("duplicate unit accepted")
	}
	gram := Revision{Name: "a", Portions: []Portion{{UnitGram, d("1")}}}
	if err := gram.Validate(); err == nil {
		t.Error("gram portion accepted")
	}
}

func TestAliases(t *testing.T) {
	got := NormalizeAliases([]string{" オート ", "", "オーツ", "オート"})
	if len(got) != 2 || got[0] != "オート" || got[1] != "オーツ" {
		t.Errorf("aliases = %q", got)
	}
	if err := (Revision{Name: "a", Aliases: []string{" "}}).Validate(); err == nil {
		t.Error("blank alias accepted")
	}
}

func TestSearchKey(t *testing.T) {
	tests := map[string]string{
		"アマランサス　玄穀":  "あまらんさす 玄穀",
		"ＣＯＣＡ  ＣＯＬＡ": "coca cola",
		"ＣＯＣＡコーラ":    "cocaこーら",
		"ﾁｷﾝ":        "ちきん",
	}
	for in, want := range tests {
		if got := SearchKey(in); got != want {
			t.Errorf("SearchKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearchTerms(t *testing.T) {
	got := SearchTerms(" めし　精白米 ")
	if len(got) != 2 || got[0] != "めし" || got[1] != "精白米" {
		t.Errorf("terms = %q", got)
	}
}

func TestBasisAmounts(t *testing.T) {
	energy := nutrient.Profile{nutrient.Energy: nutrient.Measured(d("500"))}
	unknown := Revision{Name: "バーガー", Nutrients: energy, Basis: Basis{Unit: "1個"}}
	if err := unknown.Validate(); err != nil {
		t.Fatal(err)
	}
	g, p, err := unknown.Amounts(Quantity{d("2"), "1個"})
	if err != nil || !g.IsZero() || !p.Get(nutrient.Energy).Amount.Equal(d("1000")) {
		t.Errorf("unknown weight = %s, %v, %v", g, p, err)
	}
	if _, _, err := unknown.Amounts(Quantity{d("100"), UnitGram}); !errors.Is(err, ErrUnknownMass) {
		t.Errorf("grams of unknown weight err = %v", err)
	}
	if _, ok := unknown.Per100g(); ok {
		t.Error("per 100g of unknown weight")
	}

	known := Revision{Name: "バーガー", Nutrients: energy, Basis: Basis{Unit: "1個", Grams: d("200")}, Portions: []Portion{{Unit: "半分", Grams: d("100")}}}
	if err := known.Validate(); err != nil {
		t.Fatal(err)
	}
	g, p, err = known.Amounts(Quantity{d("50"), UnitGram})
	if err != nil || !g.Equal(d("50")) || !p.Get(nutrient.Energy).Amount.Equal(d("125")) {
		t.Errorf("grams of known weight = %s, %v, %v", g, p, err)
	}
	if _, p, _ = known.Amounts(Quantity{d("1"), "半分"}); !p.Get(nutrient.Energy).Amount.Equal(d("250")) {
		t.Errorf("portion = %v", p)
	}
	if per, ok := known.Per100g(); !ok || !per.Get(nutrient.Energy).Amount.Equal(d("250")) {
		t.Errorf("per 100g = %v", per)
	}

	bad := []Revision{
		{Name: "x", Basis: Basis{Unit: "g"}},
		{Name: "x", Basis: Basis{Grams: d("10")}},
		{Name: "x", Basis: Basis{Unit: "1個"}, Portions: []Portion{{Unit: "半分", Grams: d("1")}}},
		{Name: "x", Basis: Basis{Unit: "1個", Grams: d("10")}, Portions: []Portion{{Unit: "1個", Grams: d("1")}}},
	}
	for i, r := range bad {
		if r.Validate() == nil {
			t.Errorf("bad[%d] accepted", i)
		}
	}
}
