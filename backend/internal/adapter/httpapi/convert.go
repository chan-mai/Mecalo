package httpapi

import (
	"errors"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi/gen"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/recipe"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// NOTE: 生成型とapp・domain型の相互変換のみ

func f64(d decimal.Decimal) float64 {
	v, _ := d.Float64()
	return v
}

func f64p(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	v := f64(*d)
	return &v
}

func dec(v float64) decimal.Decimal {
	return decimal.NewFromFloat(v)
}

func decp(v *float64) *decimal.Decimal {
	if v == nil {
		return nil
	}
	d := dec(*v)
	return &d
}

func toDate(d calendar.Date) openapi_types.Date {
	return openapi_types.Date{Time: d.Time()}
}

func fromDate(d openapi_types.Date) calendar.Date {
	return calendar.Of(d.Year(), d.Month(), d.Day())
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func toUser(u user.User) gen.User {
	return gen.User{Id: string(u.ID), DisplayName: u.DisplayName, Role: gen.Role(u.Role), CreatedAt: u.CreatedAt}
}

func toNutrientValue(v nutrient.Value) gen.NutrientValue {
	return gen.NutrientValue{Amount: f64(v.Amount), Status: gen.NutrientStatus(v.Status), Coverage: f64(v.Coverage)}
}

// 未計測はキーごと省略
func toProfile(p nutrient.Profile) gen.NutrientProfile {
	out := make(gen.NutrientProfile, len(p))
	for code, v := range p {
		if !v.Known() {
			continue
		}
		out[string(code)] = toNutrientValue(v)
	}
	return out
}

func fromProfile(p gen.NutrientProfile) (nutrient.Profile, error) {
	out := make(nutrient.Profile, len(p))
	for code, v := range p {
		val := nutrient.Value{Amount: dec(v.Amount), Status: nutrient.Status(v.Status), Coverage: dec(v.Coverage)}
		if err := val.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", code, err)
		}
		out[nutrient.Code(code)] = val
	}
	return out, nil
}

func fromNutrientInputs(in map[string]gen.NutrientInput) (nutrient.Profile, error) {
	out := make(nutrient.Profile, len(in))
	for code, v := range in {
		if _, ok := nutrient.Lookup(nutrient.Code(code)); !ok {
			return nil, fmt.Errorf("unknown nutrient %q", code)
		}
		status := nutrient.StatusMeasured
		if v.Status != nil {
			status = nutrient.Status(*v.Status)
		}
		switch {
		case status == nutrient.StatusMissing:
			continue
		case status == nutrient.StatusTrace:
			out[nutrient.Code(code)] = nutrient.Trace()
		case v.Amount == nil:
			continue
		case status == nutrient.StatusEstimated:
			out[nutrient.Code(code)] = nutrient.Estimated(dec(*v.Amount))
		default:
			out[nutrient.Code(code)] = nutrient.Measured(dec(*v.Amount))
		}
	}
	return out, out.Validate()
}

func toQuantity(q food.Quantity) gen.Quantity {
	return gen.Quantity{Amount: f64(q.Amount), Unit: q.Unit}
}

func fromQuantity(q gen.Quantity) food.Quantity {
	return food.Quantity{Amount: dec(q.Amount), Unit: q.Unit}
}

func toPortions(ps []food.Portion) []gen.Portion {
	out := make([]gen.Portion, 0, len(ps))
	for _, p := range ps {
		out = append(out, gen.Portion{Unit: p.Unit, Grams: f64(p.Grams)})
	}
	return out
}

func fromPortions(ps []gen.Portion) []food.Portion {
	out := make([]food.Portion, 0, len(ps))
	for _, p := range ps {
		out = append(out, food.Portion{Unit: p.Unit, Grams: dec(p.Grams)})
	}
	return out
}

func toSource(s food.Source) gen.FoodSource {
	return gen.FoodSource{DatasetId: string(s.DatasetID), Reference: s.Reference}
}

func toRevision(r food.Revision) gen.FoodRevision {
	return gen.FoodRevision{
		Id:        string(r.ID),
		Number:    int32(r.Number),
		Name:      r.Name,
		Brand:     r.Brand,
		Aliases:   nonNil(r.Aliases),
		Nutrients: toProfile(r.Nutrients),
		Portions:  toPortions(r.Portions),
		Source:    toSource(r.Source),
		CreatedAt: r.CreatedAt,
	}
}

func fromRevision(foodID food.ID, r gen.FoodRevision) (food.Revision, error) {
	p, err := fromProfile(r.Nutrients)
	if err != nil {
		return food.Revision{}, err
	}
	return food.Revision{
		ID:        food.RevisionID(r.Id),
		FoodID:    foodID,
		Number:    int(r.Number),
		Name:      r.Name,
		Brand:     r.Brand,
		Aliases:   r.Aliases,
		Nutrients: p,
		Portions:  fromPortions(r.Portions),
		Source:    food.Source{DatasetID: food.DatasetID(r.Source.DatasetId), Reference: r.Source.Reference},
		CreatedAt: r.CreatedAt,
	}, nil
}

func toFood(v app.FoodView, viewer user.ID) gen.Food {
	return gen.Food{
		Id:       string(v.Food.ID),
		Kind:     gen.FoodKind(v.Food.Kind),
		Barcode:  v.Food.Barcode,
		Editable: v.Food.EditableBy(viewer),
		Favorite: v.Favorite,
		Deleted:  v.Food.Deleted(),
		Revision: toRevision(v.Revision),
	}
}

func toFoods(vs []app.FoodView, viewer user.ID) []gen.Food {
	out := make([]gen.Food, 0, len(vs))
	for _, v := range vs {
		out = append(out, toFood(v, viewer))
	}
	return out
}

func fromFoodInput(in gen.FoodInput) (app.FoodInput, error) {
	p, err := fromNutrientInputs(in.Nutrients)
	if err != nil {
		return app.FoodInput{}, err
	}
	out := app.FoodInput{Name: in.Name, Brand: str(in.Brand), Barcode: str(in.Barcode), Nutrients: p}
	if in.Aliases != nil {
		out.Aliases = *in.Aliases
	}
	if in.Portions != nil {
		out.Portions = fromPortions(*in.Portions)
	}
	return out, nil
}

func toEntry(e meal.Entry) gen.MealEntry {
	return gen.MealEntry{
		Id:             string(e.ID),
		Date:           toDate(e.Date),
		Slot:           gen.MealSlot(e.Slot),
		FoodId:         string(e.FoodID),
		FoodRevisionId: string(e.FoodRevisionID),
		FoodName:       e.FoodName,
		Brand:          e.Brand,
		Source:         toSource(e.Source),
		Quantity:       toQuantity(e.Quantity),
		Grams:          f64(e.Grams),
		Nutrients:      toProfile(e.Nutrients),
		ClientRef:      e.ClientRef,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func toEntries(es []meal.Entry) []gen.MealEntry {
	out := make([]gen.MealEntry, 0, len(es))
	for _, e := range es {
		out = append(out, toEntry(e))
	}
	return out
}

func fromEntry(uid user.ID, e gen.MealEntry) (meal.Entry, error) {
	p, err := fromProfile(e.Nutrients)
	if err != nil {
		return meal.Entry{}, err
	}
	slot := meal.Slot(e.Slot)
	if !slot.Valid() {
		return meal.Entry{}, fmt.Errorf("invalid slot %q", e.Slot)
	}
	return meal.Entry{
		ID:             meal.ID(e.Id),
		UserID:         uid,
		Date:           fromDate(e.Date),
		Slot:           slot,
		FoodID:         food.ID(e.FoodId),
		FoodRevisionID: food.RevisionID(e.FoodRevisionId),
		FoodName:       e.FoodName,
		Brand:          e.Brand,
		Source:         food.Source{DatasetID: food.DatasetID(e.Source.DatasetId), Reference: e.Source.Reference},
		Quantity:       fromQuantity(e.Quantity),
		Grams:          dec(e.Grams),
		Nutrients:      p,
		ClientRef:      e.ClientRef,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}, nil
}

func toRecipeLines(lines []recipe.Line) []gen.RecipeLine {
	out := make([]gen.RecipeLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, gen.RecipeLine{
			FoodId: string(l.FoodID), RevisionId: string(l.RevisionID), Name: l.Name,
			Quantity: toQuantity(l.Quantity), Grams: f64(l.Grams),
		})
	}
	return out
}

func fromRecipeLines(lines []gen.RecipeLine) []recipe.Line {
	out := make([]recipe.Line, 0, len(lines))
	for _, l := range lines {
		out = append(out, recipe.Line{
			FoodID: food.ID(l.FoodId), RevisionID: food.RevisionID(l.RevisionId), Name: l.Name,
			Quantity: fromQuantity(l.Quantity), Grams: dec(l.Grams),
		})
	}
	return out
}

func toRecipe(v app.RecipeView, viewer user.ID) gen.Recipe {
	return gen.Recipe{
		Food:        toFood(v.Food, viewer),
		Lines:       toRecipeLines(v.Recipe.Lines),
		RawGrams:    f64(v.Recipe.RawGrams),
		CookedGrams: f64(v.Recipe.CookedGrams),
		Servings:    f64(v.Recipe.Servings),
	}
}

func fromRecipeInput(in gen.RecipeInput) app.RecipeInput {
	out := app.RecipeInput{Name: in.Name}
	if in.CookedGrams != nil {
		out.CookedGrams = dec(*in.CookedGrams)
	}
	if in.Servings != nil {
		out.Servings = dec(*in.Servings)
	}
	for _, ing := range in.Ingredients {
		out.Ingredients = append(out.Ingredients, app.IngredientInput{FoodID: food.ID(ing.FoodId), Quantity: fromQuantity(ing.Quantity)})
	}
	return out
}

func toTargets(ts target.Targets) gen.Targets {
	out := gen.Targets{Edition: ts.Edition, AgeBand: ts.Band, EnergyKcal: f64(ts.EnergyKcal)}
	for _, def := range nutrient.Catalog() {
		t, ok := ts.ByCode[def.Code]
		if !ok {
			continue
		}
		nt := gen.NutrientTarget{
			Code: string(t.Code),
			Ear:  f64p(t.EAR), Rda: f64p(t.RDA), Ai: f64p(t.AI), Ul: f64p(t.UL),
			DgMin: f64p(t.DGMin), DgMax: f64p(t.DGMax), Goal: f64p(t.Goal),
		}
		if t.Custom != nil {
			nt.CustomMin = f64p(t.Custom.Min)
			nt.CustomMax = f64p(t.Custom.Max)
		}
		out.Nutrients = append(out.Nutrients, nt)
	}
	return out
}

func toEvaluations(evs []target.Evaluation) []gen.Evaluation {
	out := make([]gen.Evaluation, 0, len(evs))
	for _, e := range evs {
		out = append(out, gen.Evaluation{
			Code: string(e.Code), Level: gen.Level(e.Level), Value: toNutrientValue(e.Value),
			Lower: f64p(e.Lower), Upper: f64p(e.Upper),
		})
	}
	return out
}

func toDailyReport(r app.DailyReport) gen.DailyReport {
	out := gen.DailyReport{
		Summary: gen.DaySummary{
			Date:       toDate(r.Summary.Date),
			EntryCount: int32(r.Summary.EntryCount),
			Total:      toProfile(r.Summary.Total),
			BySlot:     map[string]gen.NutrientProfile{},
		},
		Evaluations: toEvaluations(r.Evaluations),
	}
	for slot, p := range r.Summary.BySlot {
		out.Summary.BySlot[string(slot)] = toProfile(p)
	}
	if r.Targets != nil {
		t := toTargets(*r.Targets)
		out.Targets = &t
	}
	return out
}

func toRangeReport(r app.RangeReport) gen.RangeReport {
	out := gen.RangeReport{
		From:               toDate(r.From),
		To:                 toDate(r.To),
		RecordedDays:       int32(r.RecordedDays),
		Average:            toProfile(r.Average),
		AverageEvaluations: toEvaluations(r.AverageEvaluations),
	}
	for _, d := range r.Days {
		out.Days = append(out.Days, toDailyReport(d))
	}
	return out
}

func toWeight(w body.Weight) gen.Weight {
	return gen.Weight{Date: toDate(w.Date), Kg: f64(w.Kg), BodyFatPct: f64p(w.BodyFatPct)}
}

func toWeights(ws []body.Weight) []gen.Weight {
	out := make([]gen.Weight, 0, len(ws))
	for _, w := range ws {
		out = append(out, toWeight(w))
	}
	return out
}

func toSettings(s port.Settings) gen.Settings {
	p := s.Profile
	out := gen.Settings{
		EffectiveFrom: toDate(s.EffectiveFrom),
		Profile: gen.Profile{
			Sex:              gen.Sex(p.Sex),
			BirthDate:        toDate(p.BirthDate),
			HeightCm:         f64(p.HeightCm),
			Activity:         gen.ActivityLevel(p.Activity),
			Goal:             gen.Goal(p.Goal),
			EnergyAdjustKcal: f64(p.EnergyAdjustKcal),
			Menstruating:     p.Menstruating,
		},
		Overrides: []gen.TargetOverride{},
	}
	for _, def := range nutrient.Catalog() {
		if o, ok := s.Overrides[def.Code]; ok {
			out.Overrides = append(out.Overrides, gen.TargetOverride{Code: string(def.Code), Min: f64p(o.Min), Max: f64p(o.Max)})
		}
	}
	return out
}

func fromProfileInput(p gen.Profile) body.Profile {
	return body.Profile{
		Sex:              body.Sex(p.Sex),
		BirthDate:        fromDate(p.BirthDate),
		HeightCm:         dec(p.HeightCm),
		Activity:         body.ActivityLevel(p.Activity),
		Goal:             body.Goal(p.Goal),
		EnergyAdjustKcal: dec(p.EnergyAdjustKcal),
		Menstruating:     p.Menstruating,
	}
}

func fromOverrides(os []gen.TargetOverride) map[nutrient.Code]target.Override {
	out := make(map[nutrient.Code]target.Override, len(os))
	for _, o := range os {
		out[nutrient.Code(o.Code)] = target.Override{Min: decp(o.Min), Max: decp(o.Max)}
	}
	return out
}

func fromSettings(uid user.ID, s gen.Settings) port.Settings {
	p := fromProfileInput(s.Profile)
	p.UserID = uid
	return port.Settings{UserID: uid, EffectiveFrom: fromDate(s.EffectiveFrom), Profile: p, Overrides: fromOverrides(s.Overrides)}
}

func toDataset(d food.Dataset) gen.Dataset {
	return gen.Dataset{
		Id: string(d.ID), Kind: string(d.Kind), Name: d.Name, Version: d.Version, License: d.License,
		Url: d.URL, Attribution: d.Attribution, ImportedAt: d.ImportedAt,
	}
}

func toQuickPicks(q app.QuickPicks, viewer user.ID) gen.QuickPicks {
	conv := func(ps []app.QuickPick) []gen.QuickPick {
		out := make([]gen.QuickPick, 0, len(ps))
		for _, p := range ps {
			out = append(out, gen.QuickPick{Food: toFood(p.Food, viewer), Quantity: toQuantity(p.Quantity)})
		}
		return out
	}
	searches := q.Searches
	if searches == nil {
		searches = []string{}
	}
	return gen.QuickPicks{Recent: conv(q.Recent), Frequent: conv(q.Frequent), Favorites: conv(q.Favorites), Searches: searches}
}

func ToBundle(b app.Bundle) gen.ExportBundle {
	out := gen.ExportBundle{
		Format:     gen.MecaloExport,
		Version:    gen.N1,
		ExportedAt: b.ExportedAt,
		User:       toUser(b.User),
		Settings:   []gen.Settings{},
		Weights:    toWeights(b.Weights),
		Foods:      []gen.ExportFood{},
		Favorites:  []string{},
		Meals:      toEntries(b.Meals),
	}
	for _, s := range b.Settings {
		out.Settings = append(out.Settings, toSettings(s))
	}
	for _, id := range b.Favorites {
		out.Favorites = append(out.Favorites, string(id))
	}
	for _, of := range b.Foods {
		ef := gen.ExportFood{
			Id: string(of.Food.ID), Kind: gen.FoodKind(of.Food.Kind), Barcode: of.Food.Barcode,
			CreatedAt: of.Food.CreatedAt, DeletedAt: of.Food.DeletedAt,
		}
		for _, r := range of.Revisions {
			ef.Revisions = append(ef.Revisions, toRevision(r))
		}
		if len(of.Recipes) > 0 {
			m := map[string]gen.RecipeComposition{}
			for id, res := range of.Recipes {
				m[string(id)] = gen.RecipeComposition{
					Lines: toRecipeLines(res.Lines), RawGrams: f64(res.RawGrams),
					CookedGrams: f64(res.CookedGrams), Servings: f64(res.Servings),
				}
			}
			ef.Recipes = &m
		}
		out.Foods = append(out.Foods, ef)
	}
	return out
}

func FromBundle(uid user.ID, in gen.ExportBundle) (app.Bundle, error) {
	if in.Format != gen.MecaloExport || in.Version != gen.N1 {
		return app.Bundle{}, errors.New("unsupported export format")
	}
	b := app.Bundle{ExportedAt: in.ExportedAt}
	for _, s := range in.Settings {
		b.Settings = append(b.Settings, fromSettings(uid, s))
	}
	for _, w := range in.Weights {
		b.Weights = append(b.Weights, body.Weight{UserID: uid, Date: fromDate(w.Date), Kg: dec(w.Kg), BodyFatPct: decp(w.BodyFatPct)})
	}
	for _, id := range in.Favorites {
		b.Favorites = append(b.Favorites, food.ID(id))
	}
	for _, e := range in.Meals {
		entry, err := fromEntry(uid, e)
		if err != nil {
			return app.Bundle{}, err
		}
		b.Meals = append(b.Meals, entry)
	}
	for _, ef := range in.Foods {
		kind := food.Kind(ef.Kind)
		if !kind.Valid() || kind == food.KindStandard {
			return app.Bundle{}, fmt.Errorf("food %s: unsupported kind %q", ef.Id, ef.Kind)
		}
		of := app.OwnedFood{Food: food.Food{
			ID: food.ID(ef.Id), OwnerID: uid, Kind: kind, Barcode: ef.Barcode,
			CreatedAt: ef.CreatedAt, DeletedAt: ef.DeletedAt,
		}}
		for _, r := range ef.Revisions {
			rev, err := fromRevision(of.Food.ID, r)
			if err != nil {
				return app.Bundle{}, err
			}
			of.Revisions = append(of.Revisions, rev)
		}
		if ef.Recipes != nil {
			of.Recipes = map[food.RevisionID]recipe.Result{}
			for id, rc := range *ef.Recipes {
				of.Recipes[food.RevisionID(id)] = recipe.Result{
					Lines: fromRecipeLines(rc.Lines), RawGrams: dec(rc.RawGrams),
					CookedGrams: dec(rc.CookedGrams), Servings: dec(rc.Servings),
				}
			}
		}
		b.Foods = append(b.Foods, of)
	}
	return b, nil
}
