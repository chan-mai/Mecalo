package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestCozyCorner(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/product/":                  "cozycorner_product.html",
		"/product/cutcake/":          "cozycorner_cutcake.html",
		"/product/cutcake/1001.html": "cozycorner_item.html",
		"/product/gift/2002.html":    "empty.html",
	})
	s := &sink{}
	if _, err := (&CozyCorner{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 0 {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	sf := s.foods[0]
	if sf.ID != "cozycorner-1001" || sf.Name != "テストモンブラン" || sf.Basis.Unit != "1個" {
		t.Errorf("food = %+v", sf)
	}
	if sf.URL != "https://www.cozycorner.co.jp/product/cutcake/1001.html" {
		t.Errorf("url = %s", sf.URL)
	}
	checkEnergy(t, sf, "381")
	if v := sf.Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(dec("52.5")) {
		t.Errorf("carbohydrate = %s", v.Amount)
	}
}
