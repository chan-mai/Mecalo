package official

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

var doutorRoutes = map[string]string{
	"/allergy_check/doutor_coffee_shop/search/?menu_category=all": "doutor/dcs_search.html",
	"/allergy_check/excelsior_caffe/search/?menu_category=all":    "doutor/exc_search.html",
	"/allergy_check/excelsior_caffe/item/detail/21001001":         "doutor/drink.html",
	"/allergy_check/excelsior_caffe/item/detail/21002001":         "doutor/food.html",
}

func TestDoutor(t *testing.T) {
	srv, fetch := server(t, doutorRoutes)
	d := &Doutor{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := d.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	regular, large, sand := s.foods[0], s.foods[1], s.foods[2]
	if regular.ID != "doutor-group-excelsior_caffe-21001001-1" || regular.Name != "テストコーヒー Rサイズ" || regular.Brand != "エクセルシオール カフェ" {
		t.Errorf("regular = %+v", regular)
	}
	if regular.URL != doutorBase+"/allergy_check/excelsior_caffe/item/detail/21001001" {
		t.Errorf("url = %s", regular.URL)
	}
	if large.Name != "テストコーヒー Lサイズ" || large.Reference != "excelsior_caffe-21001001-2" {
		t.Errorf("large = %+v", large)
	}
	checkEnergy(t, large, "9")
	if sand.ID != "doutor-group-excelsior_caffe-21002001" || sand.Name != "テストサンド" {
		t.Errorf("sand = %+v", sand)
	}
	if v := sand.Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("1.18")) {
		t.Errorf("salt = %s", v.Amount)
	}

	// 1サイズでも取得済みなら詳細を省略
	fresh := food.SystemID("doutor-group", "excelsior_caffe-21001001-1")
	s = &sink{fresh: map[food.ID]bool{fresh: true}}
	if _, err := d.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || !slices.Equal(s.seen, []food.ID{fresh}) {
		t.Errorf("foods = %+v, seen = %v", s.foods, s.seen)
	}
}

func TestDoutorFailedItem(t *testing.T) {
	srv, fetch := server(t, doutorRoutes)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/allergy_check/excelsior_caffe/item/detail/21001001" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer broken.Close()
	s := &sink{}
	if _, err := (&Doutor{Fetch: fetch, Base: broken.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != doutorSizes+1 || !slices.Contains(s.failed, "doutor-group-excelsior_caffe-21001001-2") {
		t.Errorf("foods = %d, failed = %v", len(s.foods), s.failed)
	}
}
