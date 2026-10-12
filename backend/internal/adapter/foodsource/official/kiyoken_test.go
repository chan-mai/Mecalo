package official

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

var kiyokenRoutes = map[string]string{
	"/wp-json/wp/v2/products?per_page=100&page=1": "kiyoken/products.json",
	"/products/shiumai/test-shiumai.html":         "kiyoken/shiumai.html",
	"/products/obento/test-bento.html":            "kiyoken/bento.html",
	"/products/goods/test-goods.html":             "kiyoken/goods.html",
}

func TestKiyoken(t *testing.T) {
	srv, fetch := server(t, kiyokenRoutes)
	s := &sink{}
	if _, err := (&Kiyoken{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 || len(s.failed) != 0 {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	shiumai, bento := s.foods[0], s.foods[1]
	if shiumai.ID != "kiyoken-test-shiumai" || shiumai.Name != "テストシウマイ" || shiumai.Basis.Unit != "1個" {
		t.Errorf("shiumai = %+v", shiumai)
	}
	if shiumai.URL != "https://kiyoken.com/products/shiumai/test-shiumai.html" {
		t.Errorf("url = %s", shiumai.URL)
	}
	checkEnergy(t, shiumai, "32")
	if bento.Basis.Unit != "1包装" {
		t.Errorf("bento basis = %+v", bento.Basis)
	}
	checkEnergy(t, bento, "522")
}

func TestKiyokenFailedItem(t *testing.T) {
	srv, fetch := server(t, kiyokenRoutes)
	// 弁当の詳細は常に503
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/products/obento/test-bento.html" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer broken.Close()
	s := &sink{}
	if _, err := (&Kiyoken{Fetch: fetch, Base: broken.URL}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 1 || s.failed[0] != "kiyoken-test-bento" {
		t.Errorf("foods = %d, failed = %v", len(s.foods), s.failed)
	}
}
