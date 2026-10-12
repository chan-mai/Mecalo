package app

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 公式データ等のシステム食品の取込
type Imports struct {
	d Deps
}

// 取込元での1食品
type SystemFood = port.SystemFood

type ImportStats struct {
	Created   int
	Updated   int
	Unchanged int
}

const importBatchSize = 200

// 内容が変わった食品のみ新リビジョン追加, 既存記録不変
func (s *Imports) Upsert(ctx context.Context, ds food.Dataset, foods []SystemFood) (ImportStats, error) {
	return s.upsert(ctx, ds, foods, nil)
}

// 保存先の食品IDをseenへ通知
func (s *Imports) upsert(ctx context.Context, ds food.Dataset, foods []SystemFood, seen func(food.ID)) (ImportStats, error) {
	var stats ImportStats
	if ds.ImportedAt.IsZero() {
		ds.ImportedAt = s.d.Clock.Now()
	}
	if err := s.d.Store.Datasets().Upsert(ctx, ds); err != nil {
		return stats, err
	}
	for start := 0; start < len(foods); start += importBatchSize {
		batch := foods[start:min(start+importBatchSize, len(foods))]
		err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
			for _, sf := range batch {
				id, res, err := s.upsertOne(ctx, r, ds, sf)
				if err != nil {
					return err
				}
				if seen != nil {
					seen(id)
				}
				switch res {
				case created:
					stats.Created++
				case updated:
					stats.Updated++
				default:
					stats.Unchanged++
				}
			}
			return nil
		})
		if err != nil {
			return stats, err
		}
	}
	return stats, nil
}

type upsertResult int

const (
	unchanged upsertResult = iota
	created
	updated
)

func (s *Imports) upsertOne(ctx context.Context, r port.Repositories, ds food.Dataset, sf SystemFood) (food.ID, upsertResult, error) {
	now := s.d.Clock.Now()
	sf.Barcode, _ = food.NormalizeBarcode(sf.Barcode)
	rev := systemRevision(food.RevisionID(s.d.IDs.NewID()), ds, sf, now)
	if err := rev.Validate(); err != nil {
		return "", unchanged, invalid(err)
	}

	f, cur, err := r.Foods().Get(ctx, "", sf.ID)
	// 旧形式IDで登録済みの同一商品
	if errors.Is(err, ErrNotFound) && sf.Reference != "" {
		f, cur, err = r.Foods().FindBySource(ctx, ds.ID, sf.Reference)
	}
	if errors.Is(err, ErrNotFound) {
		f = food.Food{ID: sf.ID, Kind: sf.Kind, Barcode: sf.Barcode, CurrentRevisionID: rev.ID, CreatedAt: now}
		return f.ID, created, r.Foods().Create(ctx, f, rev)
	}
	if err != nil {
		return "", unchanged, err
	}
	if !f.IsSystem() {
		return "", unchanged, ErrConflict
	}
	if f.Discontinued() {
		if err := r.Foods().SetDiscontinued(ctx, f.ID, nil); err != nil {
			return "", unchanged, err
		}
	}
	rev.FoodID = f.ID
	if sameContent(cur, rev) {
		return f.ID, unchanged, nil
	}
	rev.Number = cur.Number + 1
	return f.ID, updated, r.Foods().AddRevision(ctx, rev)
}

func systemRevision(id food.RevisionID, ds food.Dataset, sf SystemFood, now time.Time) food.Revision {
	return food.Revision{
		ID:        id,
		FoodID:    sf.ID,
		Number:    1,
		Name:      sf.Name,
		Brand:     sf.Brand,
		Aliases:   food.NormalizeAliases(sf.Aliases),
		Nutrients: sf.Nutrients,
		Basis:     sf.Basis,
		Portions:  sf.Portions,
		Source:    food.Source{DatasetID: ds.ID, Reference: sf.Reference, URL: sf.URL, RetrievedAt: now},
		CreatedAt: now,
	}
}

// 取込前の検証
func (s *Imports) valid(ds food.Dataset, sf SystemFood) bool {
	return sf.ID != "" && sf.Reference != "" && systemRevision("", ds, sf, time.Time{}).Validate() == nil
}

func sameContent(a, b food.Revision) bool {
	if a.Name != b.Name || a.Brand != b.Brand || a.Basis.Unit != b.Basis.Unit || !a.Basis.Grams.Equal(b.Basis.Grams) || !slices.Equal(a.Aliases, b.Aliases) || len(a.Nutrients) != len(b.Nutrients) || len(a.Portions) != len(b.Portions) {
		return false
	}
	for code, va := range a.Nutrients {
		vb, ok := b.Nutrients[code]
		if !ok || va.Status != vb.Status || !va.Amount.Equal(vb.Amount) || !va.Coverage.Equal(vb.Coverage) {
			return false
		}
	}
	for i := range a.Portions {
		if a.Portions[i].Unit != b.Portions[i].Unit || !a.Portions[i].Grams.Equal(b.Portions[i].Grams) {
			return false
		}
	}
	return true
}
