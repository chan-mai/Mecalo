package app

import (
	"context"
	"errors"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type Meals struct {
	d Deps
}

type AddEntryInput struct {
	Date      calendar.Date
	Slot      meal.Slot
	FoodID    food.ID
	Quantity  food.Quantity
	ClientRef string
}

// nilの項目は変更なし
type UpdateEntryInput struct {
	Date     *calendar.Date
	Slot     *meal.Slot
	Quantity *food.Quantity
}

type CopyInput struct {
	FromDate calendar.Date
	// nilは全区分
	FromSlot *meal.Slot
	ToDate   calendar.Date
	// nilは元と同じ区分
	ToSlot *meal.Slot
}

func (s *Meals) Day(ctx context.Context, uid user.ID, date calendar.Date) ([]meal.Entry, error) {
	return s.d.Store.Meals().ListRange(ctx, uid, date, date)
}

func (s *Meals) Get(ctx context.Context, uid user.ID, id meal.ID) (meal.Entry, error) {
	return s.d.Store.Meals().Get(ctx, uid, id)
}

// ClientRefが既存なら既存記録をレスポンス
func (s *Meals) Add(ctx context.Context, uid user.ID, in AddEntryInput) (meal.Entry, error) {
	in.ClientRef = strings.TrimSpace(in.ClientRef)
	var out meal.Entry
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		if in.ClientRef != "" {
			existing, err := r.Meals().FindByClientRef(ctx, uid, in.ClientRef)
			if err == nil {
				out = existing
				return nil
			}
			if !errors.Is(err, ErrNotFound) {
				return err
			}
		}

		f, rev, err := r.Foods().Get(ctx, uid, in.FoodID)
		if err != nil {
			return err
		}
		if f.Deleted() {
			return ErrNotFound
		}
		e, err := meal.NewEntry(meal.NewEntryInput{
			ID:        meal.ID(s.d.IDs.NewID()),
			UserID:    uid,
			Date:      in.Date,
			Slot:      in.Slot,
			Quantity:  in.Quantity,
			ClientRef: in.ClientRef,
			Now:       s.d.Clock.Now(),
		}, rev)
		if err != nil {
			return invalid(err)
		}
		if err := r.Meals().Create(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	return out, err
}

// 数量変更時は記録時のリビジョンで再計算
func (s *Meals) Update(ctx context.Context, uid user.ID, id meal.ID, in UpdateEntryInput) (meal.Entry, error) {
	var out meal.Entry
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		e, err := r.Meals().Get(ctx, uid, id)
		if err != nil {
			return err
		}
		now := s.d.Clock.Now()
		if in.Quantity != nil {
			rev, err := r.Foods().GetRevision(ctx, e.FoodRevisionID)
			if err != nil {
				return err
			}
			if e, err = e.Resolve(rev, *in.Quantity, now); err != nil {
				return invalid(err)
			}
		}
		if in.Date != nil {
			e.Date = *in.Date
		}
		if in.Slot != nil {
			if !in.Slot.Valid() {
				return invalid(errors.New("invalid slot"))
			}
			e.Slot = *in.Slot
		}
		e.UpdatedAt = now
		if err := r.Meals().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	return out, err
}

func (s *Meals) Delete(ctx context.Context, uid user.ID, id meal.ID) error {
	return s.d.Store.Meals().Delete(ctx, uid, id)
}

// 食品の現行リビジョンで再計算, 削除済み食品は元の値を複製
func (s *Meals) Copy(ctx context.Context, uid user.ID, in CopyInput) ([]meal.Entry, error) {
	if in.ToSlot != nil && !in.ToSlot.Valid() {
		return nil, invalid(errors.New("invalid slot"))
	}
	var out []meal.Entry
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		src, err := r.Meals().ListRange(ctx, uid, in.FromDate, in.FromDate)
		if err != nil {
			return err
		}
		now := s.d.Clock.Now()
		for _, e := range src {
			if in.FromSlot != nil && e.Slot != *in.FromSlot {
				continue
			}
			slot := e.Slot
			if in.ToSlot != nil {
				slot = *in.ToSlot
			}
			cp, err := s.copyEntry(ctx, r, uid, e, in.ToDate, slot)
			if err != nil {
				return err
			}
			cp.CreatedAt = now
			cp.UpdatedAt = now
			if err := r.Meals().Create(ctx, cp); err != nil {
				return err
			}
			out = append(out, cp)
		}
		return nil
	})
	return out, err
}

func (s *Meals) copyEntry(ctx context.Context, r port.Repositories, uid user.ID, e meal.Entry, date calendar.Date, slot meal.Slot) (meal.Entry, error) {
	id := meal.ID(s.d.IDs.NewID())
	now := s.d.Clock.Now()
	f, rev, err := r.Foods().Get(ctx, uid, e.FoodID)
	if errors.Is(err, ErrNotFound) || (err == nil && f.Deleted()) {
		return e.Duplicate(id, date, slot, now)
	}
	if err != nil {
		return meal.Entry{}, err
	}
	cp, err := meal.NewEntry(meal.NewEntryInput{
		ID: id, UserID: uid, Date: date, Slot: slot, Quantity: quantityFor(rev, e), Now: now,
	}, rev)
	if err != nil {
		return meal.Entry{}, invalid(err)
	}
	return cp, nil
}

// 食品の現行リビジョンでスナップショットを再生成
func (s *Meals) Recalculate(ctx context.Context, uid user.ID, id meal.ID) (meal.Entry, error) {
	var out meal.Entry
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		e, err := r.Meals().Get(ctx, uid, id)
		if err != nil {
			return err
		}
		f, rev, err := r.Foods().Get(ctx, uid, e.FoodID)
		if err != nil {
			return err
		}
		if f.Deleted() {
			return ErrNotFound
		}
		if e, err = e.Resolve(rev, quantityFor(rev, e), s.d.Clock.Now()); err != nil {
			return invalid(err)
		}
		if err := r.Meals().Update(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	return out, err
}

// 新リビジョンに同じ単位がなければグラムで指定
func quantityFor(rev food.Revision, e meal.Entry) food.Quantity {
	if _, err := rev.Grams(e.Quantity); err == nil {
		return e.Quantity
	}
	return food.Quantity{Amount: e.Grams, Unit: food.UnitGram}
}

// 記録画面の候補
type QuickPicks struct {
	Recent    []QuickPick
	Frequent  []QuickPick
	Favorites []QuickPick
	Searches  []string
}

type QuickPick struct {
	Food     FoodView
	Quantity food.Quantity
}

const quickPickLimit = 20

func (s *Meals) QuickPicks(ctx context.Context, uid user.ID) (QuickPicks, error) {
	var out QuickPicks
	favs, err := (&Foods{d: s.d}).favoriteSet(ctx, uid)
	if err != nil {
		return out, err
	}
	lastQuantity := map[food.ID]food.Quantity{}

	recent, err := s.d.Store.Meals().Recent(ctx, uid, quickPickLimit)
	if err != nil {
		return out, err
	}
	for _, e := range recent {
		lastQuantity[e.FoodID] = e.Quantity
		if v, ok := s.pick(ctx, uid, e.FoodID, e.Quantity, favs); ok {
			out.Recent = append(out.Recent, v)
		}
	}

	since := today(s.d.Clock).AddDays(-frequentDays)
	usage, err := s.d.Store.Meals().Frequent(ctx, uid, since, quickPickLimit)
	if err != nil {
		return out, err
	}
	for _, u := range usage {
		lastQuantity[u.FoodID] = u.LastQuantity
		if v, ok := s.pick(ctx, uid, u.FoodID, u.LastQuantity, favs); ok {
			out.Frequent = append(out.Frequent, v)
		}
	}

	favIDs, err := s.d.Store.Favorites().List(ctx, uid)
	if err != nil {
		return out, err
	}
	for _, id := range favIDs {
		if v, ok := s.pick(ctx, uid, id, lastQuantity[id], favs); ok {
			out.Favorites = append(out.Favorites, v)
		}
	}

	out.Searches, err = s.d.Store.SearchHistory().List(ctx, uid, 10)
	return out, err
}

func (s *Meals) pick(ctx context.Context, uid user.ID, id food.ID, q food.Quantity, favs map[food.ID]bool) (QuickPick, bool) {
	f, rev, err := s.d.Store.Foods().Get(ctx, uid, id)
	if err != nil || f.Deleted() {
		return QuickPick{}, false
	}
	if q.Unit == "" {
		q = defaultQuantity(rev)
	} else if _, err := rev.Grams(q); err != nil {
		q = defaultQuantity(rev)
	}
	return QuickPick{Food: FoodView{Food: f, Revision: rev, Favorite: favs[id]}, Quantity: q}, true
}

func defaultQuantity(rev food.Revision) food.Quantity {
	if len(rev.Portions) > 0 {
		return food.Quantity{Amount: one, Unit: rev.Portions[0].Unit}
	}
	return food.Quantity{Amount: hundredGrams, Unit: food.UnitGram}
}
