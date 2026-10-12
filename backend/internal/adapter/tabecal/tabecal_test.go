package tabecal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

const sample = `{"meta":{"plan":"anonymous","total":2},"data":[
{"item_key":"0123456789abcdef","chain":"テスト食堂","name":"牛丼","size":"並盛","kcal":600,"protein_g":20.5,"fat_g":null,"carb_g":null,"fiber_g":2.0,"sugar_g":80.0,"salt_g":2.5,"sodium_mg":null,"weight_g":null},
{"item_key":"","name":"不正"}]}`

const chains = `{"data":[{"slug":"ikinari","name_ja":"いきなり！ステーキ"},{"slug":"matsuya","name_ja":"松屋"}]}`

const ikinari = `{"data":[
{"item_key":"aaaa000000000001","chain":"いきなり！ステーキ","name":"【トッピング】オニオンソース","kcal":40},
{"item_key":"aaaa000000000002","chain":"いきなり！ステーキ","name":"ワイルドステーキ","size":"300g","kcal":800}]}`

func server(t *testing.T, calls *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls = append(*calls, r.URL.Path+"?"+r.URL.RawQuery)
		if r.Header.Get("X-Api-Key") != "" {
			t.Error("key sent without configuration")
		}
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/chains":
			_, _ = w.Write([]byte(chains))
		case q.Get("chain") == "ikinari":
			_, _ = w.Write([]byte(ikinari))
		case q.Get("q") == "牛丼":
			_, _ = w.Write([]byte(sample))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFind(t *testing.T) {
	var calls []string
	srv := server(t, &calls)
	foods, err := New(srv.URL, "", "test").Find(context.Background(), "牛丼", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(foods) != 1 {
		t.Fatalf("got %d foods", len(foods))
	}
	if calls[0] != "/items?limit=50&q=%E7%89%9B%E4%B8%BC" {
		t.Errorf("first call = %s", calls[0])
	}
	f := foods[0]
	if f.ID != "tabecal-0123456789abcdef" || f.Name != "牛丼 並盛" || f.Brand != "テスト食堂" {
		t.Errorf("food = %+v", f)
	}
	if f.Basis.Unit != servingUnit || !f.Basis.Grams.IsZero() {
		t.Errorf("basis = %+v", f.Basis)
	}
	if v := f.Nutrients[nutrient.Carbohydrate]; !v.Amount.Equal(decimal.NewFromInt(82)) || v.Status != nutrient.StatusEstimated {
		t.Errorf("carbohydrate = %+v", v)
	}
	if _, ok := f.Nutrients[nutrient.Fat]; ok {
		t.Error("null fat must be missing")
	}
}

func TestFindByChainName(t *testing.T) {
	var calls []string
	srv := server(t, &calls)
	c := New(srv.URL, "", "test")
	// 記号の有無が違うチェーン名
	foods, err := c.Find(context.Background(), "いきなりステーキ ワイルド", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(foods) != 2 || foods[0].Name != "ワイルドステーキ 300g" {
		t.Errorf("foods = %+v", foods)
	}
	want := "/items?chain=ikinari&limit=50&q=%E3%82%8F%E3%81%84%E3%82%8B%E3%81%A9"
	if calls[len(calls)-1] != want {
		t.Errorf("chain call = %s, want %s", calls[len(calls)-1], want)
	}

	// チェーン一覧の再取得なし
	calls = calls[:0]
	if _, err := c.Find(context.Background(), "まつや", 50); err != nil {
		t.Fatal(err)
	}
	for _, call := range calls {
		if call == "/chains?" {
			t.Error("chains fetched again")
		}
	}
}
