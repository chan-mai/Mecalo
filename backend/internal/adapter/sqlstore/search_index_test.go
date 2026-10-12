package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type fakeReader map[string]string

func (f fakeReader) Reading(s string) string {
	if r, ok := f[s]; ok {
		return r
	}
	return s
}

func TestReindexOnReaderChange(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	s, err := OpenSQLite(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	ds := food.Dataset{ID: "d", Kind: food.DatasetMEXT, Name: "d", ImportedAt: time.Now()}
	rev := food.Revision{ID: "r", FoodID: "f", Number: 1, Name: "にんじん", Source: food.Source{DatasetID: "d"}, CreatedAt: time.Now()}
	if err := s.Datasets().Upsert(ctx, ds); err != nil {
		t.Fatal(err)
	}
	if err := s.Foods().Create(ctx, food.Food{ID: "f", Kind: food.KindStandard, CreatedAt: time.Now()}, rev); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = OpenSQLite(ctx, path, WithReader(fakeReader{"にんじん": "ニンジン"}))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hits, err := s.Foods().Search(ctx, port.FoodSearchQuery{Terms: [][]string{{"にんじん"}}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %+v", hits)
	}
	var reading string
	if err := s.q.queryRow(ctx, "SELECT reading_key FROM food_revisions WHERE id = 'r'").Scan(&reading); err != nil {
		t.Fatal(err)
	}
	if reading != "にんじん" {
		t.Errorf("reading_key = %q", reading)
	}
}

func TestCompactKeys(t *testing.T) {
	k := indexer{reader: fakeReader{}}.keys("バニラ クリーム フラペチーノ Tall", "スターバックス", []string{"バニラ フラペ"})
	if k.compact != "ばにらくりーむふらぺちーのtall すたーばっくす ばにらふらぺ" {
		t.Errorf("compact = %q", k.compact)
	}
	if k.compactReading != "ばにらくりーむふらぺちーのtall ばにらふらぺ" {
		t.Errorf("compact reading = %q", k.compactReading)
	}
}

func TestSearchAcrossWordBoundary(t *testing.T) {
	ctx := context.Background()
	s, err := OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ds := food.Dataset{ID: "d", Kind: food.DatasetOfficial, Name: "d", ImportedAt: time.Now()}
	rev := food.Revision{ID: "r", FoodID: "f", Number: 1, Name: "バニラ クリーム フラペチーノ Tall", Brand: "スターバックス", Source: food.Source{DatasetID: "d"}, CreatedAt: time.Now()}
	if err := s.Datasets().Upsert(ctx, ds); err != nil {
		t.Fatal(err)
	}
	if err := s.Foods().Create(ctx, food.Food{ID: "f", Kind: food.KindProduct, CreatedAt: time.Now()}, rev); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"ばにらくりーむふらぺちーの", "くりーむふらぺ"} {
		hits, err := s.Foods().Search(ctx, port.FoodSearchQuery{Terms: [][]string{{q}}, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 1 {
			t.Errorf("%s: hits = %d", q, len(hits))
		}
	}
	// 名前とブランドをまたぐ一致は対象外
	hits, err := s.Foods().Search(ctx, port.FoodSearchQuery{Terms: [][]string{{"tallすたー"}}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Errorf("cross-field hits = %d", len(hits))
	}
}
