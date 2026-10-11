package food

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
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
