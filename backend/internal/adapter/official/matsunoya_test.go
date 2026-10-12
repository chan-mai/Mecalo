package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestMatsunoya(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/matsunoya/safety/allergen.html":             "matsunoya_list.html",
		"/matsunoya/menu/teishoku/tei_test_rosu.html": "matsunoya_item.html",
	})
	s := &sink{}
	if _, err := (&Matsunoya{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 0 {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	sf := s.foods[0]
	if sf.ID != "matsunoya-tei_test_rosu" || sf.Name != "テストロースかつ定食" || sf.Basis.Unit != "1食" {
		t.Errorf("food = %+v", sf)
	}
	if sf.URL != "https://www.matsuyafoods.co.jp/matsunoya/menu/teishoku/tei_test_rosu.html" {
		t.Errorf("url = %s", sf.URL)
	}
	checkEnergy(t, sf, "1000")
	for code, want := range map[nutrient.Code]string{
		nutrient.Protein: "40.5", nutrient.Fat: "50", nutrient.Carbohydrate: "100", nutrient.SaltEquivalent: "5.1",
	} {
		if v := sf.Nutrients[code]; !v.Amount.Equal(dec(want)) {
			t.Errorf("%s = %s, want %s", code, v.Amount, want)
		}
	}
}
