package official

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestSaintMarc(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/allergy_check/brand/04-O-A/search/?allergy[]=&contain=0": "saintmarc_search.html",
		"/allergy_check/brand/12-B-A/search/?allergy[]=&contain=0": "saintmarc_kura_search.html",
		"/allergy_check/brand/04-O-A/detail/00011":                 "saintmarc_item.html",
	})
	// 2件目の詳細は常に503
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/allergy_check/brand/04-O-A/detail/00012" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer broken.Close()
	sm := &SaintMarc{Fetch: fetch, Base: broken.URL}
	s := &sink{}
	if _, err := sm.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 1 || s.failed[0] != "saint-marc-04-O-A-00012" {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	sugar := s.foods[0]
	if sugar.ID != "saint-marc-04-O-A-00011" || sugar.Name != "テストシュガー" || sugar.Brand != "サンマルクカフェ" {
		t.Errorf("sugar = %+v", sugar)
	}
	if sugar.URL != saintMarcBase+"/allergy_check/brand/04-O-A/detail/00011" {
		t.Errorf("url = %s", sugar.URL)
	}
	checkEnergy(t, sugar, "11")
	if v, ok := sugar.Nutrients[nutrient.SaltEquivalent]; !ok || !v.Amount.IsZero() {
		t.Errorf("salt = %+v", v)
	}

	s = &sink{fresh: map[food.ID]bool{"saint-marc-04-O-A-00011": true}}
	if _, err := sm.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 0 || len(s.seen) != 1 {
		t.Errorf("foods = %+v, seen = %v", s.foods, s.seen)
	}
}
