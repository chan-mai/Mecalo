package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestSushiro(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/menu/?pref=01&mode=service": "sushiro/pref01.html",
		"/menu/?pref=04&mode=service": "sushiro/pref04.html",
		"/menu/?pref=13&mode=service": "sushiro/pref13.html",
		"/menu/menu_detail/?s_id=101": "sushiro/menu_101.html",
		"/menu/menu_detail/?s_id=201": "sushiro/menu_201.html",
	})
	sr := &Sushiro{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := sr.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	tuna, juice, kanto := s.foods[0], s.foods[1], s.foods[2]
	if tuna.ID != "sushiro-1001" || tuna.Name != "テスト まぐろ" || tuna.Basis.Unit != "1食" {
		t.Errorf("tuna = %+v", tuna)
	}
	if tuna.URL != "https://www.akindo-sushiro.co.jp/menu/menu_detail/?s_id=101" {
		t.Errorf("url = %s", tuna.URL)
	}
	checkEnergy(t, tuna, "53")
	if juice.ID != "sushiro-1003" || juice.Basis.PerUnit() {
		t.Errorf("juice = %+v", juice)
	}
	checkEnergy(t, juice, "45")
	if kanto.ID != "sushiro-2001" {
		t.Errorf("kanto = %+v", kanto)
	}
	checkEnergy(t, kanto, "206")
}

func TestSushiroNoStores(t *testing.T) {
	srv, fetch := server(t, map[string]string{})
	sr := &Sushiro{Fetch: fetch, Base: srv.URL}
	if _, err := sr.Sync(context.Background(), port.CatalogState{}, &sink{}); err == nil {
		t.Error("want error")
	}
}
