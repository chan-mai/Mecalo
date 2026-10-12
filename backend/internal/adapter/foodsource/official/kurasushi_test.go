package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestKuraSushi(t *testing.T) {
	srv, fetch := server(t, map[string]string{"/menu/": "kurasushi/index.html"})
	ks := &KuraSushi{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := ks.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 2 {
		t.Fatalf("foods = %+v", s.foods)
	}
	tuna, custard := s.foods[0], s.foods[1]
	if tuna.ID != "kurasushi-000001" || tuna.Name != "テスト まぐろ" || tuna.Basis.Unit != "1食" {
		t.Errorf("tuna = %+v", tuna)
	}
	checkEnergy(t, tuna, "82")
	checkEnergy(t, custard, "9.6")
}
