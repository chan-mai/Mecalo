// 材料と完成重量からの栄養値算出
package recipe

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

const ServingUnit = "人前"

var hundred = decimal.NewFromInt(100)

type Ingredient struct {
	Revision food.Revision
	Quantity food.Quantity
}

type Definition struct {
	Ingredients []Ingredient
	// 0は材料重量の合計を使用
	CookedGrams decimal.Decimal
	// 0は人前単位なし
	Servings decimal.Decimal
}

// 保存用の材料行
type Line struct {
	FoodID     food.ID
	RevisionID food.RevisionID
	Name       string
	Quantity   food.Quantity
	Grams      decimal.Decimal
}

type Result struct {
	Lines       []Line
	RawGrams    decimal.Decimal
	CookedGrams decimal.Decimal
	Servings    decimal.Decimal
	// 完成品100gあたり
	Per100g  nutrient.Profile
	Portions []food.Portion
}

func Compute(def Definition) (Result, error) {
	if len(def.Ingredients) == 0 {
		return Result{}, errors.New("recipe needs at least one ingredient")
	}
	if def.CookedGrams.IsNegative() || def.Servings.IsNegative() {
		return Result{}, errors.New("cooked grams and servings must not be negative")
	}

	lines := make([]Line, 0, len(def.Ingredients))
	parts := make([]nutrient.Part, 0, len(def.Ingredients))
	raw := decimal.Zero
	for i, ing := range def.Ingredients {
		grams, values, err := ing.Revision.Amounts(ing.Quantity)
		if err == nil && grams.IsZero() {
			err = food.ErrUnknownMass
		}
		if err != nil {
			return Result{}, fmt.Errorf("ingredient %d: %w", i+1, err)
		}
		raw = raw.Add(grams)
		parts = append(parts, nutrient.Part{Grams: grams, Values: values})
		lines = append(lines, Line{
			FoodID:     ing.Revision.FoodID,
			RevisionID: ing.Revision.ID,
			Name:       ing.Revision.Name,
			Quantity:   ing.Quantity,
			Grams:      grams,
		})
	}

	cooked := def.CookedGrams
	if cooked.IsZero() {
		cooked = raw
	}

	res := Result{
		Lines:       lines,
		RawGrams:    raw,
		CookedGrams: cooked,
		Servings:    def.Servings,
		Per100g:     nutrient.Sum(parts).Scale(hundred.Div(cooked)),
	}
	if def.Servings.IsPositive() {
		res.Portions = []food.Portion{{Unit: ServingUnit, Grams: cooked.Div(def.Servings)}}
	}
	return res, nil
}
