package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestBigBoy(t *testing.T) {
	srv, fetch := server(t, map[string]string{"/allergen/menu/json/menu.json": "bigboy/menu.json"})
	s := &sink{}
	state, err := (&BigBoy{Fetch: fetch, Base: srv.URL}).Sync(context.Background(), port.CatalogState{}, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 5 || state.Cursor == "" {
		t.Fatalf("foods = %+v, state = %+v", s.foods, state)
	}
	// 掲載順, 文字列のメニューID
	soup, burger, large := s.foods[0], s.foods[1], s.foods[2]
	if soup.ID != "bigboy-200a" || burger.ID != "bigboy-100" || burger.Brand != "ビッグボーイ" || burger.Basis.Unit != "1食" {
		t.Errorf("menus = %+v, %+v", soup, burger)
	}
	checkEnergy(t, burger, "549")
	if _, ok := large.Nutrients[nutrient.Protein]; ok {
		t.Error("'-' protein must be missing")
	}
	// 複数メニューに掲載のソースは1件
	sauce := s.foods[3]
	if sauce.ID != "bigboy-sauce-900" || sauce.Name != "テストソース(テストハンバーグ用)" || s.foods[4].Reference != "sauce-901" {
		t.Errorf("sauces = %+v, %+v", sauce, s.foods[4])
	}
	checkEnergy(t, s.foods[4], "44")
}
