package official

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestTopvalu(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/items/list/100000000?page=1": "topvalu/list.html",
		"/items/detail/4549414000000":  "topvalu/item.html",
		"/items/detail/4549414999999":  "empty.html",
	})
	tv := &Topvalu{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := tv.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 {
		t.Fatalf("foods = %+v", s.foods)
	}
	sf := s.foods[0]
	if sf.ID != "topvalu-4549414000000" || sf.Barcode != "4549414000000" || sf.Name != "テスト 酢だこ" {
		t.Errorf("food = %+v", sf)
	}
	if sf.Basis.Unit != "10g" || !sf.Basis.Grams.Equal(dec("10")) {
		t.Errorf("basis = %+v", sf.Basis)
	}
	if len(sf.Portions) != 1 || sf.Portions[0].Unit != "全量" || !sf.Portions[0].Grams.Equal(dec("80")) {
		t.Errorf("portions = %+v", sf.Portions)
	}
	checkEnergy(t, sf, "9")
	if v := sf.Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("0.1")) {
		t.Errorf("salt = %s", v.Amount)
	}

	if len(s.failed) != 0 {
		t.Errorf("failed = %v", s.failed)
	}

	// 詳細取得済みは一覧での確認のみ
	s = &sink{fresh: map[food.ID]bool{"topvalu-4549414000000": true}}
	if _, err := tv.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 0 || len(s.seen) != 1 {
		t.Errorf("fresh sync foods=%d seen=%v", len(s.foods), s.seen)
	}
}

func TestTopvaluValidRevision(t *testing.T) {
	doc, err := extract.ParseHTML([]byte(`<h1 class="item-detail__name">x</h1><dl><dt>栄養成分</dt><dd>1本(500ml)当たり エネルギー0kcal</dd></dl><dl><dt>規格：</dt><dd>500ml</dd></dl>`))
	if err != nil {
		t.Fatal(err)
	}
	sf, ok := parseTopvalu(doc, "4549414000001")
	if !ok {
		t.Fatal("not parsed")
	}
	rev := food.Revision{Name: sf.Name, Nutrients: sf.Nutrients, Basis: sf.Basis, Portions: sf.Portions}
	if err := rev.Validate(); err != nil {
		t.Errorf("revision invalid: %v (%+v)", err, sf)
	}
}

func TestTopvaluFailedItem(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/items/list/100000000?page=1": "topvalu/list.html",
		"/items/detail/4549414000000":  "topvalu/item.html",
	})
	// 2件目の詳細は常に503
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/items/detail/4549414999999" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer broken.Close()
	s := &sink{}
	if _, err := (&Topvalu{Fetch: fetch, Base: broken.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 1 || s.failed[0] != "topvalu-4549414999999" {
		t.Errorf("foods = %d, failed = %v", len(s.foods), s.failed)
	}
}
