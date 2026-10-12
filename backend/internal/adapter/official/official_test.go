package official

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

type sink struct {
	foods  []app.SystemFood
	fresh  map[food.ID]bool
	seen   []food.ID
	failed []food.ID
}

func (s *sink) Emit(sf app.SystemFood) error { s.foods = append(s.foods, sf); return nil }
func (s *sink) Fresh(id food.ID) bool        { return s.fresh[id] }
func (s *sink) Seen(id food.ID) error        { s.seen = append(s.seen, id); return nil }
func (s *sink) Failed(id food.ID, _ error) error {
	s.failed = append(s.failed, id)
	return nil
}

// パスとtestdataのファイルの対応で応答
func server(t *testing.T, routes map[string]string) (*httptest.Server, *webfetch.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		name, ok := routes[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c := webfetch.New("Mecalo-Agent/vtest")
	c.Interval = 0
	c.RetryDelay = 0
	return srv, c
}

func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func checkEnergy(t *testing.T, sf app.SystemFood, want string) {
	t.Helper()
	if v := sf.Nutrients[nutrient.Energy]; !v.Amount.Equal(dec(want)) {
		t.Errorf("%s energy = %s, want %s", sf.Name, v.Amount, want)
	}
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
