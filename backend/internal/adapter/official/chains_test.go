package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestMcDonalds(t *testing.T) {
	srv, fetch := server(t, map[string]string{mcdonaldsPath: "mcdonalds.html"})
	s := &sink{}
	state, err := (&McDonalds{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 || state.Cursor == "" {
		t.Fatalf("foods = %+v, state = %+v", s.foods, state)
	}
	if s.foods[0].ID != "mcdonalds-1210" || s.foods[1].ID != "mcdonalds-5000" || s.foods[2].ID != "mcdonalds-5000-テストポテト M" {
		t.Errorf("ids = %s %s %s", s.foods[0].ID, s.foods[1].ID, s.foods[2].ID)
	}
	checkEnergy(t, s.foods[0], "524")
	if v := s.foods[0].Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("2.7")) {
		t.Errorf("salt = %s", v.Amount)
	}
}

func TestStarbucks(t *testing.T) {
	srv, fetch := server(t, map[string]string{"/allergy/json/nutritions.json": "starbucks.json"})
	s := &sink{}
	if _, err := (&Starbucks{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 {
		t.Fatalf("foods = %+v", s.foods)
	}
	latte, scone := s.foods[0], s.foods[1]
	if latte.Reference != "4500000000001-tall" || latte.Name != "テスト ラテ Tall" || latte.Basis.Unit != "1杯" {
		t.Errorf("latte = %+v", latte)
	}
	checkEnergy(t, latte, "223.4")
	if _, ok := latte.Nutrients[nutrient.Sodium]; ok {
		t.Error("'-' sodium must be missing")
	}
	if scone.Reference != "4500000000002" || scone.Basis.Unit != "1個" {
		t.Errorf("scone = %+v", scone)
	}
}

func TestMos(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/common/menu/js/nutrition.js":    "mos_nutrition.js",
		"/data/menu/menu_category/1.json": "mos_1.json",
		"/data/menu/menu_category/7.json": "mos_7.json",
	})
	s := &sink{}
	if _, err := (&Mos{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	burger := s.foods[0]
	if !burger.Basis.Grams.Equal(dec("219")) {
		t.Errorf("basis = %+v", burger.Basis)
	}
	if v := burger.Nutrients[nutrient.VitaminA]; !v.Amount.Equal(dec("6")) {
		t.Errorf("vitamin a = %s", v.Amount)
	}
	if s.foods[1].Name != "テストポテト S" || s.foods[2].Name != "テストポテト L" {
		t.Errorf("sizes = %s, %s", s.foods[1].Name, s.foods[2].Name)
	}
	checkEnergy(t, s.foods[2], "400")
}
