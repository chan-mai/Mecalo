package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestHorai551(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/menu/":         "horai551_menu.html",
		"/menu/butaman/": "horai551_butaman.html",
		"/menu/冷麺/":      "horai551_reimen.html",
	})
	s := &sink{}
	if _, err := (&Horai551{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 || len(s.failed) != 0 {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	butaman, reimen := s.foods[0], s.foods[1]
	if butaman.ID != "horai551-butaman" || butaman.Name != "テスト豚まん" || butaman.URL != "https://www.551horai.co.jp/menu/butaman/" {
		t.Errorf("butaman = %+v", butaman)
	}
	if butaman.Basis.Unit != "1個" || !butaman.Basis.Grams.Equal(dec("130")) {
		t.Errorf("butaman basis = %+v", butaman.Basis)
	}
	checkEnergy(t, butaman, "292")
	if v := butaman.Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(dec("41.1")) {
		t.Errorf("carbohydrate = %s", v.Amount)
	}
	if reimen.ID != "horai551-冷麺" || reimen.URL != "https://www.551horai.co.jp/menu/%E5%86%B7%E9%BA%BA/" {
		t.Errorf("reimen = %+v", reimen)
	}
	if reimen.Basis.Unit != "1食" || !reimen.Basis.Grams.Equal(dec("400")) {
		t.Errorf("reimen basis = %+v", reimen.Basis)
	}
}
