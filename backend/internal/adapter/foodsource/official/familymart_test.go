package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestFamilyMart(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/goods/safety.html":            "familymart/index.html",
		"/goods/safety/goods020.html":   "familymart/goods020.html",
		"/goods/safety/friedfoods.html": "familymart/friedfoods.html",
	})
	fm := &FamilyMart{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := fm.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	if nolink := s.foods[2]; nolink.ID != "familymart-0000003" || nolink.URL != "" {
		t.Errorf("no link food = %+v", nolink)
	}
	bento := s.foods[0]
	if bento.ID != "familymart-0000001" || bento.Name != "テスト海苔弁当" || bento.Basis.Unit != "1食" {
		t.Errorf("bento = %+v", bento)
	}
	checkEnergy(t, bento, "857")
	if v := bento.Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("4.5")) {
		t.Errorf("salt = %s", v.Amount)
	}
	chicken := s.foods[1]
	checkEnergy(t, chicken, "251.7")
	if _, ok := chicken.Nutrients[nutrient.Protein]; ok {
		t.Error("'-' must be missing")
	}
}
