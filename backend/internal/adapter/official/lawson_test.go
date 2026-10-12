package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestLawson(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/recommend/original/select/index.html":        "lawson_select.html",
		"/recommend/original/bento/":                   "lawson_bento.html",
		"/recommend/original/detail/1000001_1996.html": "lawson_item1.html",
		"/recommend/original/detail/1000002_1996.html": "lawson_item2.html",
	})
	lw := &Lawson{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := lw.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 {
		t.Fatalf("foods = %+v", s.foods)
	}
	salad, bento := s.foods[0], s.foods[1]
	if salad.ID != "lawson-1000001_1996" || salad.Name != "テスト ミックスサラダ" || salad.Basis.PerUnit() {
		t.Errorf("salad = %+v", salad)
	}
	checkEnergy(t, salad, "21")
	if v := salad.Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(dec("5")) {
		t.Errorf("carbohydrate = %s", v.Amount)
	}
	if bento.Basis.Unit != "1個" || !bento.Basis.Grams.Equal(dec("450")) {
		t.Errorf("bento basis = %+v", bento.Basis)
	}
}
