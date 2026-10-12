package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestSevenEleven(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/products/a/onigiri/":               "seven_onigiri.html",
		"/products/a/onigiri/area/kanto/":    "seven_onigiri_kanto.html",
		"/products/a/onigiri/area/hokkaido/": "seven_onigiri_hokkaido.html",
		"/products/a/item/000001/kanto/":     "seven_item_000001.html",
		"/products/a/item/000001/hokkaido/":  "seven_item_000001_hokkaido.html",
		"/products/a/item/000002/":           "empty.html",
		"/products/a/item/000003/kanto/":     "seven_item_000003.html",
		"/products/a/item/000003/hokkaido/":  "seven_item_000003.html",
	})
	se := &SevenEleven{Fetch: fetch, Base: srv.URL}
	s := &sink{}
	if _, err := se.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 3 {
		t.Fatalf("foods = %+v", s.foods)
	}
	// 地域で値が異なる商品は地域を併記
	hokkaido, kanto, tuna := s.foods[0], s.foods[1], s.foods[2]
	if hokkaido.ID != "seven-eleven-000001-hokkaido" || hokkaido.Name != "テスト 手巻おにぎり 紅しゃけ(北海道)" {
		t.Errorf("hokkaido = %+v", hokkaido)
	}
	checkEnergy(t, hokkaido, "180")
	if kanto.ID != "seven-eleven-000001-kanto" || kanto.Name != "テスト 手巻おにぎり 紅しゃけ(関東)" || kanto.Basis.Unit != "1食" {
		t.Errorf("kanto = %+v", kanto)
	}
	if kanto.URL != "https://www.sej.co.jp/products/a/item/000001/kanto/" {
		t.Errorf("url = %s", kanto.URL)
	}
	checkEnergy(t, kanto, "174")
	if v := kanto.Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(dec("35.9")) {
		t.Errorf("carbohydrate = %s", v.Amount)
	}
	// 全地域で同じ値は1件
	if tuna.ID != "seven-eleven-000003" || tuna.Name != "テスト 手巻おにぎり ツナマヨ" {
		t.Errorf("tuna = %+v", tuna)
	}

	// 地域別に保存済みの商品は一覧での確認のみ
	s = &sink{fresh: map[food.ID]bool{"seven-eleven-000001-kanto": true, "seven-eleven-000001-hokkaido": true, "seven-eleven-000003": true}}
	if _, err := se.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 0 || len(s.seen) != 3 {
		t.Errorf("fresh sync foods=%d seen=%v", len(s.foods), s.seen)
	}
}
