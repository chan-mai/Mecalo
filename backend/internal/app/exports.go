package app

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/recipe"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type Exports struct {
	d Deps
}

// 1利用者分の全データ
type Bundle struct {
	ExportedAt time.Time
	User       user.User
	Settings   []port.Settings
	Weights    []body.Weight
	Foods      []OwnedFood
	Favorites  []food.ID
	Meals      []meal.Entry
}

type OwnedFood struct {
	Food      food.Food
	Revisions []food.Revision
	// recipe種別のみ, キーはリビジョンID
	Recipes map[food.RevisionID]recipe.Result
}

var (
	minDate = calendar.Of(1900, time.January, 1)
	maxDate = calendar.Of(9999, time.December, 31)
)

func (s *Exports) Export(ctx context.Context, u user.User) (Bundle, error) {
	b := Bundle{ExportedAt: s.d.Clock.Now(), User: u}
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		var err error
		if b.Settings, err = r.Settings().List(ctx, u.ID); err != nil {
			return err
		}
		if b.Weights, err = r.Weights().ListRange(ctx, u.ID, minDate, maxDate); err != nil {
			return err
		}
		if b.Favorites, err = r.Favorites().List(ctx, u.ID); err != nil {
			return err
		}
		if b.Meals, err = r.Meals().ListRange(ctx, u.ID, minDate, maxDate); err != nil {
			return err
		}
		owned, err := r.Foods().ListOwned(ctx, u.ID)
		if err != nil {
			return err
		}
		for _, h := range owned {
			of, err := ownedFood(ctx, r, u.ID, h.Food)
			if err != nil {
				return err
			}
			b.Foods = append(b.Foods, of)
		}
		return nil
	})
	return b, err
}

func ownedFood(ctx context.Context, r port.Repositories, uid user.ID, f food.Food) (OwnedFood, error) {
	revs, err := r.Foods().ListRevisions(ctx, uid, f.ID)
	if err != nil {
		return OwnedFood{}, err
	}
	of := OwnedFood{Food: f, Revisions: revs}
	if f.Kind != food.KindRecipe {
		return of, nil
	}
	of.Recipes = map[food.RevisionID]recipe.Result{}
	for _, rev := range revs {
		res, err := r.Recipes().Get(ctx, rev.ID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return OwnedFood{}, err
		}
		of.Recipes[rev.ID] = res
	}
	return of, nil
}

type ImportResult struct {
	Foods    int
	Meals    int
	Weights  int
	Settings int
	Skipped  int
}

// 既存IDは保持したまま追加, 同一IDがあれば読み飛ばし
func (s *Exports) Import(ctx context.Context, uid user.ID, b Bundle) (ImportResult, error) {
	var res ImportResult
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		if err := r.Datasets().Upsert(ctx, UserDataset); err != nil {
			return err
		}
		for _, of := range b.Foods {
			imported, err := importFood(ctx, r, uid, of)
			if err != nil {
				return err
			}
			if imported {
				res.Foods++
			} else {
				res.Skipped++
			}
		}
		for _, e := range b.Meals {
			_, err := r.Meals().Get(ctx, uid, e.ID)
			if err == nil {
				res.Skipped++
				continue
			}
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			e.UserID = uid
			if err := r.Meals().Create(ctx, e); err != nil {
				return err
			}
			res.Meals++
		}
		for _, w := range b.Weights {
			w.UserID = uid
			if err := w.Validate(); err != nil {
				return invalid(err)
			}
			if err := r.Weights().Save(ctx, w); err != nil {
				return err
			}
			res.Weights++
		}
		for _, st := range b.Settings {
			st.UserID = uid
			st.Profile.UserID = uid
			if err := st.Profile.Validate(); err != nil {
				return invalid(err)
			}
			if err := r.Settings().Save(ctx, st); err != nil {
				return err
			}
			res.Settings++
		}
		for _, id := range b.Favorites {
			if _, _, err := r.Foods().Get(ctx, uid, id); err != nil {
				continue
			}
			if err := r.Favorites().Add(ctx, uid, id, s.d.Clock.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	return res, err
}

func importFood(ctx context.Context, r port.Repositories, uid user.ID, of OwnedFood) (bool, error) {
	if len(of.Revisions) == 0 {
		return false, invalid(errors.New("food without revisions"))
	}
	_, _, err := r.Foods().Get(ctx, uid, of.Food.ID)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return false, err
	}

	revs := append([]food.Revision(nil), of.Revisions...)
	sort.Slice(revs, func(i, j int) bool { return revs[i].Number < revs[j].Number })
	for _, rev := range revs {
		if err := rev.Validate(); err != nil {
			return false, invalid(err)
		}
	}
	f := of.Food
	f.OwnerID = uid
	f.CurrentRevisionID = revs[0].ID
	if err := r.Foods().Create(ctx, f, revs[0]); err != nil {
		return false, err
	}
	for _, rev := range revs[1:] {
		if err := r.Foods().AddRevision(ctx, rev); err != nil {
			return false, err
		}
	}
	for revID, rec := range of.Recipes {
		if err := r.Recipes().Save(ctx, revID, rec); err != nil {
			return false, err
		}
	}
	if f.DeletedAt != nil {
		if err := r.Foods().SoftDelete(ctx, f.ID, *f.DeletedAt); err != nil {
			return false, err
		}
	}
	return true, nil
}
