package official

import (
	"context"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestTablePages(t *testing.T) {
	srv, fetch := server(t, map[string]string{
		"/quality/allergy-nutrition_value/": "ringerhut.html",
		"/menu_list/info/nutrition/27186":   "ootoya.html",
	})
	rh := RingerHut(fetch)
	rh.Base = srv.URL
	s := &sink{}
	if _, err := rh.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	// 商品名の見出しが別の行にある表, 同名は1件
	if len(s.foods) != 1 || s.foods[0].Name != "テストちゃんぽん" || s.foods[0].Brand != "リンガーハット" {
		t.Fatalf("ringerhut = %+v", s.foods)
	}
	if v := s.foods[0].Nutrients[nutrient.SaltEquivalent]; !v.Amount.Equal(dec("8.4")) {
		t.Errorf("salt = %s", v.Amount)
	}

	ot := Ootoya(fetch)
	ot.Base = srv.URL
	s = &sink{}
	if _, err := ot.Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 {
		t.Fatalf("ootoya = %+v", s.foods)
	}
	// 値に単位付き, 炭水化物は糖質と食物繊維から
	checkEnergy(t, s.foods[0], "1178")
	if v := s.foods[0].Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(dec("135.7")) {
		t.Errorf("carbohydrate = %s", v.Amount)
	}
}

func TestSkylark(t *testing.T) {
	foods, err := parseSkylark(readTestdata(t, "skylark.json"), "gusto", "ガスト")
	if err != nil {
		t.Fatal(err)
	}
	if len(foods) != 3 {
		t.Fatalf("foods = %+v", foods)
	}
	if foods[0].ID != "skylark-gusto-1_100" || foods[0].Brand != "ガスト" {
		t.Errorf("first = %+v", foods[0])
	}
	if foods[1].Name != "テストパスタ レギュラー" || foods[2].Name != "テストパスタ 大盛" || foods[2].Reference != "gusto-1_300-b" {
		t.Errorf("variations = %+v, %+v", foods[1], foods[2])
	}
	checkEnergy(t, foods[2], "800")
}

func TestSaizeriya(t *testing.T) {
	srv, fetch := server(t, map[string]string{"/view?lg=ja": "saizeriya.json"})
	s := &sink{}
	if _, err := (&Saizeriya{Fetch: fetch, URL: srv.URL + "/view?lg=ja"}).Sync(context.Background(), port.CatalogState{}, s); err != nil {
		t.Fatal(err)
	}
	if len(s.foods) != 1 || s.foods[0].ID != "saizeriya-abc" {
		t.Fatalf("foods = %+v", s.foods)
	}
	checkEnergy(t, s.foods[0], "196")
}
