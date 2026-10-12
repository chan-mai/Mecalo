package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestKourakuen(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/grandmenu/":                   "kourakuen/list.html",
		"/grandmenu/?gid=01":            "kourakuen/list_gid01.html",
		"/grandmenu/info.html?id=10001": "kourakuen/item.html",
		"/grandmenu/info.html?id=10002": "empty.html",
	})
	k := &Kourakuen{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := k.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || len(s.failed) != 0 {
		t.Fatalf("foods = %+v, failed = %v", s.foods, s.failed)
	}
	sf := s.foods[0]
	if sf.ID != "kourakuen-10001" || sf.Name != "テスト中華そば" || sf.Basis.Unit != "1食" {
		t.Errorf("food = %+v", sf)
	}
	if sf.URL != "https://stores.kourakuen.co.jp/grandmenu/info.html?id=10001" {
		t.Errorf("url = %s", sf.URL)
	}
	checkEnergy(t, sf, "639")
	if v := sf.Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("9.8")) {
		t.Errorf("salt = %s", v.Amount)
	}

	// 詳細取得済みは一覧での確認のみ
	s = &sink{fresh: map[food.ID]bool{"kourakuen-10001": true}}
	if _, err := k.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 0 || len(s.seen) != 1 {
		t.Errorf("fresh sync foods=%d seen=%v", len(s.foods), s.seen)
	}
}
