package tools

import (
	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
)

// 検索結果と日ごとの集計に含める栄養素
var mainNutrients = []nutrient.Code{
	nutrient.Energy, nutrient.Protein, nutrient.Fat, nutrient.Carbohydrate, nutrient.SaltEquivalent,
}

type amount struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
	// 値が判明している重量の割合, 1は省略
	Coverage *float64 `json:"coverage,omitempty"`
}

type basisView struct {
	Unit string `json:"unit"`
	// 1単位の重量, 不明は省略
	Grams *float64 `json:"grams,omitempty"`
}

type foodSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Brand string `json:"brand,omitempty"`
	// nilは100gあたり
	Basis *basisView `json:"basis,omitempty"`
	// 記録に使える単位
	Units     []string          `json:"units"`
	Nutrients map[string]amount `json:"nutrients"`
}

type portionView struct {
	Unit  string  `json:"unit"`
	Grams float64 `json:"grams"`
}

type foodDetail struct {
	foodSummary
	Portions []portionView `json:"portions,omitempty"`
}

type entryView struct {
	ID        string            `json:"id"`
	Date      string            `json:"date"`
	Slot      string            `json:"slot"`
	FoodID    string            `json:"food_id"`
	FoodName  string            `json:"food_name"`
	Brand     string            `json:"brand,omitempty"`
	Amount    float64           `json:"amount"`
	Unit      string            `json:"unit"`
	Grams     float64           `json:"grams"`
	Nutrients map[string]amount `json:"nutrients"`
}

type evaluationView struct {
	Code   string   `json:"code"`
	Level  string   `json:"level"`
	Amount *amount  `json:"amount,omitempty"`
	Lower  *float64 `json:"lower,omitempty"`
	Upper  *float64 `json:"upper,omitempty"`
}

type dayView struct {
	Date    string            `json:"date"`
	Today   string            `json:"today"`
	Entries []entryView       `json:"entries"`
	Total   map[string]amount `json:"total"`
	// 身体情報・体重が未登録ならfalse
	HasTargets  bool             `json:"has_targets"`
	Evaluations []evaluationView `json:"evaluations,omitempty"`
}

type daySummaryView struct {
	Date       string            `json:"date"`
	EntryCount int               `json:"entry_count"`
	Total      map[string]amount `json:"total,omitempty"`
}

type weightView struct {
	Date       string   `json:"date"`
	Kg         float64  `json:"kg"`
	BodyFatPct *float64 `json:"body_fat_pct,omitempty"`
}

type rangeView struct {
	From         string            `json:"from"`
	To           string            `json:"to"`
	Today        string            `json:"today"`
	RecordedDays int               `json:"recorded_days"`
	Average      map[string]amount `json:"average"`
	// 期間末日の目標に対する平均値の評価
	AverageEvaluations []evaluationView `json:"average_evaluations,omitempty"`
	Days               []daySummaryView `json:"days"`
	Weights            []weightView     `json:"weights"`
}

func round(d decimal.Decimal) float64 {
	return d.Round(2).InexactFloat64()
}

func roundp(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	v := round(*d)
	return &v
}

func unitOf(code nutrient.Code) string {
	def, _ := nutrient.Lookup(code)
	return string(def.Unit)
}

// 未計測は除外
func toAmount(code nutrient.Code, v nutrient.Value) (amount, bool) {
	if !v.Known() {
		return amount{}, false
	}
	a := amount{Amount: round(v.Amount), Unit: unitOf(code)}
	if v.Coverage.LessThan(decimal.NewFromInt(1)) {
		c := v.Coverage.Round(2).InexactFloat64()
		a.Coverage = &c
	}
	return a, true
}

// codes未指定は全栄養素
func toNutrients(p nutrient.Profile, codes ...nutrient.Code) map[string]amount {
	if len(codes) == 0 {
		for _, def := range nutrient.Catalog() {
			codes = append(codes, def.Code)
		}
	}
	out := map[string]amount{}
	for _, c := range codes {
		if a, ok := toAmount(c, p.Get(c)); ok {
			out[string(c)] = a
		}
	}
	return out
}

func toFoodSummary(v app.FoodView) foodSummary {
	rev := v.Revision
	out := foodSummary{
		ID:        string(v.Food.ID),
		Name:      rev.Name,
		Brand:     rev.Brand,
		Nutrients: toNutrients(rev.Nutrients, mainNutrients...),
	}
	if rev.Basis.PerUnit() {
		out.Basis = &basisView{Unit: rev.Basis.Unit}
		if rev.Basis.Grams.IsPositive() {
			out.Basis.Grams = roundp(&rev.Basis.Grams)
			out.Units = append(out.Units, food.UnitGram)
		}
		out.Units = append(out.Units, rev.Basis.Unit)
	} else {
		out.Units = append(out.Units, food.UnitGram)
	}
	for _, p := range rev.Portions {
		out.Units = append(out.Units, p.Unit)
	}
	return out
}

func toFoodDetail(v app.FoodView) foodDetail {
	out := foodDetail{foodSummary: toFoodSummary(v)}
	out.Nutrients = toNutrients(v.Revision.Nutrients)
	for _, p := range v.Revision.Portions {
		out.Portions = append(out.Portions, portionView{Unit: p.Unit, Grams: round(p.Grams)})
	}
	return out
}

func toEntryView(e meal.Entry) entryView {
	return entryView{
		ID:        string(e.ID),
		Date:      e.Date.String(),
		Slot:      string(e.Slot),
		FoodID:    string(e.FoodID),
		FoodName:  e.FoodName,
		Brand:     e.Brand,
		Amount:    round(e.Quantity.Amount),
		Unit:      e.Quantity.Unit,
		Grams:     round(e.Grams),
		Nutrients: toNutrients(e.Nutrients, mainNutrients...),
	}
}

func toEvaluations(evs []target.Evaluation) []evaluationView {
	out := make([]evaluationView, 0, len(evs))
	for _, e := range evs {
		if e.Level == target.LevelNone {
			continue
		}
		v := evaluationView{Code: string(e.Code), Level: string(e.Level), Lower: roundp(e.Lower), Upper: roundp(e.Upper)}
		if a, ok := toAmount(e.Code, e.Value); ok {
			v.Amount = &a
		}
		out = append(out, v)
	}
	return out
}

func toWeightView(w body.Weight) weightView {
	return weightView{Date: w.Date.String(), Kg: round(w.Kg), BodyFatPct: roundp(w.BodyFatPct)}
}

func toRangeView(a *app.App, r app.RangeReport, weights []body.Weight) rangeView {
	out := rangeView{
		From:               r.From.String(),
		To:                 r.To.String(),
		Today:              a.Today().String(),
		RecordedDays:       r.RecordedDays,
		Average:            toNutrients(r.Average),
		AverageEvaluations: toEvaluations(r.AverageEvaluations),
		Days:               make([]daySummaryView, 0, len(r.Days)),
		Weights:            make([]weightView, 0, len(weights)),
	}
	for _, d := range r.Days {
		v := daySummaryView{Date: d.Summary.Date.String(), EntryCount: d.Summary.EntryCount}
		if d.Summary.EntryCount > 0 {
			v.Total = toNutrients(d.Summary.Total, mainNutrients...)
		}
		out.Days = append(out.Days, v)
	}
	for _, w := range weights {
		out.Weights = append(out.Weights, toWeightView(w))
	}
	return out
}
