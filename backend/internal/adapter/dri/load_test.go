package dri

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/data"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
)

func TestLoadBundled(t *testing.T) {
	ref, err := Load(data.DRI2025)
	if err != nil {
		t.Fatal(err)
	}
	if len(ref.Bands) != 12 {
		t.Errorf("bands = %d", len(ref.Bands))
	}

	ts, err := target.Build(ref, target.Person{
		Sex: body.SexMale, Age: 35, WeightKg: decimal.NewFromInt(70), Activity: body.ActivityModerate,
	}, decimal.Zero, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 22.5 * 70 * 1.75
	if !ts.EnergyKcal.Equal(decimal.RequireFromString("2756.25")) {
		t.Errorf("energy = %s", ts.EnergyKcal)
	}
	for _, code := range []nutrient.Code{nutrient.Protein, nutrient.Fat, nutrient.Carbohydrate, nutrient.Fiber, nutrient.SaltEquivalent, nutrient.Iron, nutrient.VitaminC} {
		if _, ok := ts.ByCode[code]; !ok {
			t.Errorf("missing target for %s", code)
		}
	}
	if ts.ByCode[nutrient.Fat].DGMax == nil {
		t.Error("fat energy ratio not converted")
	}
}
