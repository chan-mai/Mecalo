package official

import (
	"context"
	"slices"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestGodiva(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/drink/":                          "godiva/drink.html",
		"/drink/clr_test_milk.html":        "godiva/clr_test_milk.html",
		"/drink/cacaofruitjuice_test.html": "godiva/cacaofruitjuice_test.html",
	})
	gd := &Godiva{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := gd.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	regular, large, juice := s.foods[0], s.foods[1], s.foods[2]
	if regular.ID != "godiva-clr_test_milk-1" || regular.Name != "テスト ミルクチョコレート ショコリキサー レギュラー" || regular.Basis.Unit != "1杯" {
		t.Errorf("regular = %+v", regular)
	}
	if regular.URL != "https://www.godiva.co.jp/drink/clr_test_milk.html" {
		t.Errorf("url = %s", regular.URL)
	}
	checkEnergy(t, regular, "358")
	if large.ID != "godiva-clr_test_milk-2" || large.Name != "テスト ミルクチョコレート ショコリキサー ラージ" {
		t.Errorf("large = %+v", large)
	}
	checkEnergy(t, large, "492")
	if juice.Name != "テスト ジュース レギュラー" {
		t.Errorf("juice = %+v", juice)
	}
}

func TestGodivaFresh(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/drink/":                   "godiva/drink.html",
		"/drink/clr_test_milk.html": "godiva/clr_test_milk.html",
	})
	gd := &Godiva{Fetch: fetch, Base: srv.URL}
	fresh := food.SystemID("godiva", "cacaofruitjuice_test-1")
	s := &sink{fresh: map[food.ID]bool{fresh: true}}
	if _, err := gd.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 || !slices.Equal(s.seen, []food.ID{fresh}) {
		t.Errorf("foods = %+v, seen = %v", s.foods, s.seen)
	}
}
