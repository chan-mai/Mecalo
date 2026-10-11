package recipe

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func gram(s string) food.Quantity {
	return food.Quantity{Amount: d(s), Unit: food.UnitGram}
}

func TestComputeCurry(t *testing.T) {
	pork := food.Revision{ID: "r-pork", FoodID: "pork", Name: "豚肉", Nutrients: nutrient.Profile{
		nutrient.Energy:     nutrient.Measured(d("200")),
		nutrient.VitaminB12: nutrient.Measured(d("0.5")),
	}}
	onion := food.Revision{ID: "r-onion", FoodID: "onion", Name: "たまねぎ",
		Portions: []food.Portion{{Unit: "個", Grams: d("200")}},
		Nutrients: nutrient.Profile{
			nutrient.Energy: nutrient.Measured(d("30")),
		}}
	roux := food.Revision{ID: "r-roux", FoodID: "roux", Name: "カレールウ", Nutrients: nutrient.Profile{
		nutrient.Energy:     nutrient.Measured(d("500")),
		nutrient.VitaminB12: nutrient.Missing(),
	}}

	res, err := Compute(Definition{
		Ingredients: []Ingredient{
			{pork, gram("300")},
			{onion, food.Quantity{Amount: d("1"), Unit: "個"}},
			{roux, gram("100")},
		},
		CookedGrams: d("1200"),
		Servings:    d("4"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// (600 + 60 + 500) kcal / 1200g * 100
	energy := res.Per100g.Get(nutrient.Energy)
	if !energy.Amount.Round(4).Equal(d("96.6667")) {
		t.Errorf("energy per 100g = %s", energy.Amount)
	}
	if !energy.Coverage.Equal(d("1")) {
		t.Errorf("energy coverage = %s", energy.Coverage)
	}

	b12 := res.Per100g.Get(nutrient.VitaminB12)
	if !b12.Amount.Mul(d("12")).Round(4).Equal(d("1.5")) {
		t.Errorf("b12 per 100g = %s", b12.Amount)
	}
	// 豚肉300g / 材料600g
	if !b12.Coverage.Equal(d("0.5")) {
		t.Errorf("b12 coverage = %s", b12.Coverage)
	}

	if !res.RawGrams.Equal(d("600")) {
		t.Errorf("raw grams = %s", res.RawGrams)
	}
	if len(res.Portions) != 1 || !res.Portions[0].Grams.Equal(d("300")) {
		t.Errorf("portions = %+v", res.Portions)
	}
	if res.Lines[1].RevisionID != "r-onion" || !res.Lines[1].Grams.Equal(d("200")) {
		t.Errorf("line = %+v", res.Lines[1])
	}
}

func TestComputeDefaultsCookedToRaw(t *testing.T) {
	salad := food.Revision{ID: "r", FoodID: "f", Name: "レタス", Nutrients: nutrient.Profile{
		nutrient.Energy: nutrient.Measured(d("10")),
	}}
	res, err := Compute(Definition{Ingredients: []Ingredient{{salad, gram("250")}}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CookedGrams.Equal(d("250")) || !res.Per100g.Get(nutrient.Energy).Amount.Equal(d("10")) {
		t.Errorf("unexpected result %+v", res)
	}
	if len(res.Portions) != 0 {
		t.Errorf("portions = %+v", res.Portions)
	}
}

func TestComputeRejectsEmpty(t *testing.T) {
	if _, err := Compute(Definition{}); err == nil {
		t.Error("empty recipe accepted")
	}
}
