package target

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func p(s string) *decimal.Decimal {
	v := d(s)
	return &v
}

// 計算確認用の架空の基準値
func fixture() Reference {
	return Reference{
		Edition: "test",
		Bands:   []AgeBand{{"18-29", 18, 30}, {"30-49", 30, 50}, {"75+", 75, 0}},
		BMRPerKg: map[body.Sex]map[string]decimal.Decimal{
			body.SexMale:   {"30-49": d("22")},
			body.SexFemale: {"30-49": d("21")},
		},
		PAL: map[string]map[body.ActivityLevel]decimal.Decimal{
			"30-49": {body.ActivityLow: d("1.5"), body.ActivityModerate: d("1.75"), body.ActivityHigh: d("2")},
		},
		Nutrients: map[nutrient.Code]NutrientReference{
			nutrient.Protein: {Values: map[Group]map[string]Indicators{
				GroupMale: {"30-49": {EAR: p("50"), RDA: p("65"), DGEnergyPctMin: p("13"), DGEnergyPctMax: p("20")}},
			}},
			nutrient.Fat: {Values: map[Group]map[string]Indicators{
				GroupMale: {"30-49": {DGEnergyPctMin: p("20"), DGEnergyPctMax: p("30")}},
			}},
			nutrient.Iron: {Values: map[Group]map[string]Indicators{
				GroupFemale:             {"30-49": {EAR: p("5"), RDA: p("6"), UL: p("40")}},
				GroupFemaleMenstruating: {"30-49": {EAR: p("9"), RDA: p("10.5"), UL: p("40")}},
			}},
			nutrient.SaltEquivalent: {Values: map[Group]map[string]Indicators{
				GroupMale: {"30-49": {DGMax: p("7.5")}},
			}},
		},
	}
}

func TestBandFor(t *testing.T) {
	ref := fixture()
	if b, err := ref.BandFor(80); err != nil || b.Label != "75+" {
		t.Errorf("band = %v, %v", b, err)
	}
	if _, err := ref.BandFor(10); err == nil {
		t.Error("age without band accepted")
	}
}

func TestBuild(t *testing.T) {
	ts, err := Build(fixture(), Person{
		Sex: body.SexMale, Age: 35, WeightKg: d("70"), Activity: body.ActivityModerate,
	}, d("-300"), map[nutrient.Code]Override{
		nutrient.VitaminC: {Min: p("200")},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 22 * 70 * 1.75 - 300
	if !ts.EnergyKcal.Equal(d("2395")) {
		t.Errorf("energy = %s", ts.EnergyKcal)
	}

	protein := ts.ByCode[nutrient.Protein]
	// 2395 * 13% / 4
	if !protein.DGMin.Round(4).Equal(d("77.8375")) {
		t.Errorf("protein dg min = %s", protein.DGMin)
	}
	fat := ts.ByCode[nutrient.Fat]
	// 2395 * 30% / 9
	if !fat.DGMax.Round(2).Equal(d("79.83")) {
		t.Errorf("fat dg max = %s", fat.DGMax)
	}
	if ts.ByCode[nutrient.VitaminC].Custom == nil {
		t.Error("override not applied")
	}
}

func TestBuildIronByMenstruation(t *testing.T) {
	base := Person{Sex: body.SexFemale, Age: 35, WeightKg: d("50"), Activity: body.ActivityLow}
	ts, err := Build(fixture(), base, decimal.Zero, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ts.ByCode[nutrient.Iron].RDA.Equal(d("6")) {
		t.Errorf("iron rda = %s", ts.ByCode[nutrient.Iron].RDA)
	}

	base.Menstruating = true
	ts, err = Build(fixture(), base, decimal.Zero, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ts.ByCode[nutrient.Iron].RDA.Equal(d("10.5")) {
		t.Errorf("iron rda = %s", ts.ByCode[nutrient.Iron].RDA)
	}
}

func TestEvaluate(t *testing.T) {
	iron := Target{Code: nutrient.Iron, Indicators: Indicators{EAR: p("5"), RDA: p("6"), UL: p("40")}}
	salt := Target{Code: nutrient.SaltEquivalent, Indicators: Indicators{DGMax: p("7.5")}}
	energy := Target{Code: nutrient.Energy, Goal: p("2000")}
	custom := Target{Code: nutrient.Iron, Indicators: iron.Indicators, Custom: &Override{Min: p("3")}}

	partial := func(amount, coverage string) nutrient.Value {
		return nutrient.Value{Amount: d(amount), Status: nutrient.StatusCalculated, Coverage: d(coverage)}
	}

	tests := []struct {
		name   string
		target Target
		value  nutrient.Value
		want   Level
	}{
		{"deficient", iron, nutrient.Measured(d("4")), LevelDeficient},
		{"low", iron, nutrient.Measured(d("5.5")), LevelLow},
		{"adequate", iron, nutrient.Measured(d("7")), LevelAdequate},
		{"excessive", iron, nutrient.Measured(d("41")), LevelExcessive},
		{"missing", iron, nutrient.Missing(), LevelUnknown},
		{"low coverage hides deficiency", iron, partial("2", "0.5"), LevelUnknown},
		{"low coverage still detects excess", iron, partial("45", "0.5"), LevelExcessive},
		{"salt over goal", salt, nutrient.Measured(d("9")), LevelHigh},
		{"salt under goal", salt, nutrient.Measured(d("5")), LevelAdequate},
		{"upper only ignores ear", Target{Code: nutrient.SaltEquivalent, Indicators: Indicators{EAR: p("1.5"), DGMax: p("6.5")}}, nutrient.Measured(d("0")), LevelAdequate},
		{"energy low", energy, nutrient.Measured(d("1700")), LevelLow},
		{"energy ok", energy, nutrient.Measured(d("2100")), LevelAdequate},
		{"energy high", energy, nutrient.Measured(d("2300")), LevelHigh},
		{"custom replaces reference", custom, nutrient.Measured(d("4")), LevelAdequate},
		{"no reference", Target{Code: nutrient.Biotin}, nutrient.Measured(d("10")), LevelNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(tt.target, tt.value).Level; got != tt.want {
				t.Errorf("level = %s, want %s", got, tt.want)
			}
		})
	}
}
