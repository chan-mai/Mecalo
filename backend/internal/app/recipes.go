package app

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/recipe"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type Recipes struct {
	d Deps
}

type RecipeInput struct {
	Name        string
	Ingredients []IngredientInput
	CookedGrams decimal.Decimal
	Servings    decimal.Decimal
}

type IngredientInput struct {
	FoodID   food.ID
	Quantity food.Quantity
}

type RecipeView struct {
	Food   FoodView
	Recipe recipe.Result
}

func (s *Recipes) Create(ctx context.Context, uid user.ID, in RecipeInput) (RecipeView, error) {
	var out RecipeView
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		res, err := s.compute(ctx, r, uid, in)
		if err != nil {
			return err
		}
		now := s.d.Clock.Now()
		f := food.Food{
			ID:        food.ID(s.d.IDs.NewID()),
			OwnerID:   uid,
			Kind:      food.KindRecipe,
			CreatedAt: now,
		}
		rev := s.revision(f.ID, 1, in.Name, res)
		if err := rev.Validate(); err != nil {
			return invalid(err)
		}
		f.CurrentRevisionID = rev.ID
		if err := r.Datasets().Upsert(ctx, UserDataset); err != nil {
			return err
		}
		if err := r.Foods().Create(ctx, f, rev); err != nil {
			return err
		}
		if err := r.Recipes().Save(ctx, rev.ID, res); err != nil {
			return err
		}
		out = RecipeView{Food: FoodView{Food: f, Revision: rev}, Recipe: res}
		return nil
	})
	return out, err
}

// 材料の現行リビジョンで再計算し新リビジョンとして保存
func (s *Recipes) Update(ctx context.Context, uid user.ID, id food.ID, in RecipeInput) (RecipeView, error) {
	var out RecipeView
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		f, cur, err := r.Foods().Get(ctx, uid, id)
		if err != nil {
			return err
		}
		if f.Kind != food.KindRecipe || !f.EditableBy(uid) || f.Deleted() {
			return ErrForbidden
		}
		res, err := s.compute(ctx, r, uid, in)
		if err != nil {
			return err
		}
		rev := s.revision(f.ID, cur.Number+1, in.Name, res)
		if err := rev.Validate(); err != nil {
			return invalid(err)
		}
		if err := r.Foods().AddRevision(ctx, rev); err != nil {
			return err
		}
		if err := r.Recipes().Save(ctx, rev.ID, res); err != nil {
			return err
		}
		f.CurrentRevisionID = rev.ID
		out = RecipeView{Food: FoodView{Food: f, Revision: rev}, Recipe: res}
		return nil
	})
	return out, err
}

func (s *Recipes) Get(ctx context.Context, uid user.ID, id food.ID) (RecipeView, error) {
	f, rev, err := s.d.Store.Foods().Get(ctx, uid, id)
	if err != nil {
		return RecipeView{}, err
	}
	if f.Kind != food.KindRecipe {
		return RecipeView{}, ErrNotFound
	}
	res, err := s.d.Store.Recipes().Get(ctx, rev.ID)
	if err != nil {
		return RecipeView{}, err
	}
	res.Per100g = rev.Nutrients
	res.Portions = rev.Portions
	return RecipeView{Food: FoodView{Food: f, Revision: rev}, Recipe: res}, nil
}

func (s *Recipes) compute(ctx context.Context, r port.Repositories, uid user.ID, in RecipeInput) (recipe.Result, error) {
	def := recipe.Definition{CookedGrams: in.CookedGrams, Servings: in.Servings}
	for _, ing := range in.Ingredients {
		f, rev, err := r.Foods().Get(ctx, uid, ing.FoodID)
		if err != nil {
			return recipe.Result{}, err
		}
		if f.Deleted() {
			return recipe.Result{}, ErrNotFound
		}
		def.Ingredients = append(def.Ingredients, recipe.Ingredient{Revision: rev, Quantity: ing.Quantity})
	}
	res, err := recipe.Compute(def)
	if err != nil {
		return recipe.Result{}, invalid(err)
	}
	return res, nil
}

func (s *Recipes) revision(id food.ID, number int, name string, res recipe.Result) food.Revision {
	return food.Revision{
		ID:        food.RevisionID(s.d.IDs.NewID()),
		FoodID:    id,
		Number:    number,
		Name:      strings.TrimSpace(name),
		Nutrients: res.Per100g,
		Portions:  res.Portions,
		Source:    food.Source{DatasetID: UserDataset.ID},
		CreatedAt: s.d.Clock.Now(),
	}
}
