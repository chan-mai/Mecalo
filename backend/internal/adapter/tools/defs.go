package tools

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

const (
	searchLimitDefault = 10
	searchLimitMax     = 30
	// 応答の大きさの上限
	maxRangeDays = 31
)

var all = []*Tool{
	define("search_foods", user.ScopeRead,
		"Search foods by name. Use the returned id to record a meal. Nutrients are per 100 g, or per one basis unit when basis is set.",
		searchFoods,
		property("limit", between(1, searchLimitMax))),
	define("get_food", user.ScopeRead,
		"Get a food with all nutrient values and the units it can be recorded in.",
		getFood),
	define("get_day", user.ScopeRead,
		"Get the meals recorded on a day, the nutrient totals and how they compare with the user's targets.",
		getDay,
		property("date", dateFormat)),
	define("get_range", user.ScopeRead,
		"Get daily totals, the average over recorded days, target evaluations of the average and body weights for a period of up to 31 days.",
		getRange,
		property("from", dateFormat), property("to", dateFormat)),
	define("add_meal", user.ScopeWrite,
		"Record a food eaten. Find the food id with search_foods first. The unit must be g or one of the food's units.",
		addMeal,
		property("date", dateFormat), property("slot", enum(meal.Slots()...)), property("amount", positive)),
	define("update_meal", user.ScopeWrite,
		"Change the date, meal slot or quantity of a recorded meal. Omitted fields are kept.",
		updateMeal,
		property("date", dateFormat), property("slot", enum(meal.Slots()...)), property("amount", positive)),
	define("delete_meal", user.ScopeWrite,
		"Delete a recorded meal.",
		deleteMeal,
		destructive),
	define("record_weight", user.ScopeWrite,
		"Record body weight for a day. An existing record on the same day is replaced.",
		recordWeight,
		property("date", dateFormat), property("kg", positive), property("body_fat_pct", between(0, 100))),
}

type searchInput struct {
	Query string `json:"query" jsonschema:"food name, separate words with spaces"`
	Limit *int   `json:"limit,omitempty" jsonschema:"maximum number of results, default 10"`
}

func searchFoods(ctx context.Context, a *app.App, uid user.ID, in searchInput) (any, error) {
	limit := searchLimitDefault
	if in.Limit != nil {
		limit = *in.Limit
	}
	views, err := a.Foods.Search(ctx, uid, in.Query, nil, limit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]foodSummary, 0, len(views))
	for _, v := range views {
		out = append(out, toFoodSummary(v))
	}
	return map[string]any{"foods": out}, nil
}

type idInput struct {
	ID string `json:"id"`
}

func getFood(ctx context.Context, a *app.App, uid user.ID, in idInput) (any, error) {
	v, err := a.Foods.Get(ctx, uid, food.ID(in.ID))
	if err != nil {
		return nil, err
	}
	return toFoodDetail(v), nil
}

type dayInput struct {
	Date string `json:"date,omitempty" jsonschema:"YYYY-MM-DD, default today"`
}

func getDay(ctx context.Context, a *app.App, uid user.ID, in dayInput) (any, error) {
	date, err := dateOr(a, in.Date)
	if err != nil {
		return nil, err
	}
	entries, err := a.Meals.Day(ctx, uid, date)
	if err != nil {
		return nil, err
	}
	rep, err := a.Reports.Daily(ctx, uid, date)
	if err != nil {
		return nil, err
	}
	out := dayView{
		Date:        date.String(),
		Today:       a.Today().String(),
		Entries:     make([]entryView, 0, len(entries)),
		Total:       toNutrients(rep.Summary.Total),
		Evaluations: toEvaluations(rep.Evaluations),
		HasTargets:  rep.Targets != nil,
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, toEntryView(e))
	}
	return out, nil
}

type rangeInput struct {
	From string `json:"from" jsonschema:"first day, YYYY-MM-DD"`
	To   string `json:"to" jsonschema:"last day, YYYY-MM-DD"`
}

func getRange(ctx context.Context, a *app.App, uid user.ID, in rangeInput) (any, error) {
	from, err := parseDate(in.From)
	if err != nil {
		return nil, err
	}
	to, err := parseDate(in.To)
	if err != nil {
		return nil, err
	}
	if !to.Before(from) && from.DaysUntil(to) >= maxRangeDays {
		return nil, app.Invalid(errors.New("range must be 31 days or less"))
	}
	rep, err := a.Reports.Range(ctx, uid, from, to)
	if err != nil {
		return nil, err
	}
	weights, err := a.Weights.Range(ctx, uid, from, to)
	if err != nil {
		return nil, err
	}
	return toRangeView(a, rep, weights), nil
}

type addMealInput struct {
	Date   string  `json:"date,omitempty" jsonschema:"YYYY-MM-DD, default today"`
	Slot   string  `json:"slot"`
	FoodID string  `json:"food_id"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit,omitempty" jsonschema:"g (default) or a unit listed for the food"`
}

func addMeal(ctx context.Context, a *app.App, uid user.ID, in addMealInput) (any, error) {
	date, err := dateOr(a, in.Date)
	if err != nil {
		return nil, err
	}
	e, err := a.Meals.Add(ctx, uid, app.AddEntryInput{
		Date:     date,
		Slot:     meal.Slot(in.Slot),
		FoodID:   food.ID(in.FoodID),
		Quantity: quantity(in.Amount, in.Unit),
	})
	if err != nil {
		return nil, err
	}
	return toEntryView(e), nil
}

type updateMealInput struct {
	ID     string   `json:"id"`
	Date   string   `json:"date,omitempty" jsonschema:"YYYY-MM-DD"`
	Slot   string   `json:"slot,omitempty"`
	Amount *float64 `json:"amount,omitempty"`
	Unit   string   `json:"unit,omitempty" jsonschema:"unit of amount, default the current unit"`
}

func updateMeal(ctx context.Context, a *app.App, uid user.ID, in updateMealInput) (any, error) {
	id := meal.ID(in.ID)
	var upd app.UpdateEntryInput
	if in.Date != "" {
		d, err := parseDate(in.Date)
		if err != nil {
			return nil, err
		}
		upd.Date = &d
	}
	if in.Slot != "" {
		s := meal.Slot(in.Slot)
		upd.Slot = &s
	}
	if in.Amount != nil {
		unit := in.Unit
		if unit == "" {
			cur, err := a.Meals.Get(ctx, uid, id)
			if err != nil {
				return nil, err
			}
			unit = cur.Quantity.Unit
		}
		q := quantity(*in.Amount, unit)
		upd.Quantity = &q
	} else if in.Unit != "" {
		return nil, app.Invalid(errors.New("unit requires amount"))
	}
	e, err := a.Meals.Update(ctx, uid, id, upd)
	if err != nil {
		return nil, err
	}
	return toEntryView(e), nil
}

func deleteMeal(ctx context.Context, a *app.App, uid user.ID, in idInput) (any, error) {
	if err := a.Meals.Delete(ctx, uid, meal.ID(in.ID)); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": in.ID}, nil
}

type weightInput struct {
	Date       string   `json:"date,omitempty" jsonschema:"YYYY-MM-DD, default today"`
	Kg         float64  `json:"kg"`
	BodyFatPct *float64 `json:"body_fat_pct,omitempty" jsonschema:"body fat percentage"`
}

func recordWeight(ctx context.Context, a *app.App, uid user.ID, in weightInput) (any, error) {
	date, err := dateOr(a, in.Date)
	if err != nil {
		return nil, err
	}
	w := body.Weight{Date: date, Kg: decimal.NewFromFloat(in.Kg)}
	if in.BodyFatPct != nil {
		v := decimal.NewFromFloat(*in.BodyFatPct)
		w.BodyFatPct = &v
	}
	saved, err := a.Weights.Save(ctx, uid, w)
	if err != nil {
		return nil, err
	}
	return toWeightView(saved), nil
}

func parseDate(s string) (calendar.Date, error) {
	d, err := calendar.Parse(s)
	if err != nil {
		return calendar.Date{}, app.Invalid(err)
	}
	return d, nil
}

// 空は当日
func dateOr(a *app.App, s string) (calendar.Date, error) {
	if s == "" {
		return a.Today(), nil
	}
	return parseDate(s)
}

func quantity(amount float64, unit string) food.Quantity {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = food.UnitGram
	}
	return food.Quantity{Amount: decimal.NewFromFloat(amount), Unit: unit}
}
