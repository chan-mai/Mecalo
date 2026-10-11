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
