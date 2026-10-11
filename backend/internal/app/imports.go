package app

import (
	"context"
	"errors"
	"slices"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 公式データ等のシステム食品の取込
type Imports struct {
	d Deps
}

// 取込元での1食品
type SystemFood struct {
	ID        food.ID
	Kind      food.Kind
	Barcode   string
	Name      string
	Brand     string
	Aliases   []string
	Reference string
	Nutrients nutrient.Profile
	Portions  []food.Portion
}

type ImportStats struct {
	Created   int
	Updated   int
	Unchanged int
}

const importBatchSize = 200

// 内容が変わった食品のみ新リビジョン追加, 既存記録不変
func (s *Imports) Upsert(ctx context.Context, ds food.Dataset, foods []SystemFood) (ImportStats, error) {
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
				res, err := s.upsertOne(ctx, r, ds, sf)
				if err != nil {
					return err
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

func (s *Imports) upsertOne(ctx context.Context, r port.Repositories, ds food.Dataset, sf SystemFood) (upsertResult, error) {
	now := s.d.Clock.Now()
	rev := food.Revision{
		ID:        food.RevisionID(s.d.IDs.NewID()),
		FoodID:    sf.ID,
		Number:    1,
		Name:      sf.Name,
		Brand:     sf.Brand,
		Aliases:   food.NormalizeAliases(sf.Aliases),
		Nutrients: sf.Nutrients,
		Portions:  sf.Portions,
		Source:    food.Source{DatasetID: ds.ID, Reference: sf.Reference},
		CreatedAt: now,
	}
	if err := rev.Validate(); err != nil {
		return unchanged, invalid(err)
	}

	f, cur, err := r.Foods().Get(ctx, "", sf.ID)
	if errors.Is(err, ErrNotFound) {
		f = food.Food{ID: sf.ID, Kind: sf.Kind, Barcode: sf.Barcode, CurrentRevisionID: rev.ID, CreatedAt: now}
		return created, r.Foods().Create(ctx, f, rev)
	}
	if err != nil {
		return unchanged, err
	}
	if !f.IsSystem() {
		return unchanged, ErrConflict
	}
	if sameContent(cur, rev) {
		return unchanged, nil
	}
	rev.Number = cur.Number + 1
	return updated, r.Foods().AddRevision(ctx, rev)
}

func sameContent(a, b food.Revision) bool {
	if a.Name != b.Name || a.Brand != b.Brand || !slices.Equal(a.Aliases, b.Aliases) || len(a.Nutrients) != len(b.Nutrients) || len(a.Portions) != len(b.Portions) {
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
