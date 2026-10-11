package openfoodfacts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

const sample = `{"code":"4901234567894","product_name":"Green tea","product_name_ja":"緑茶","brands":"Example, Other",
"serving_quantity":"500","countries_tags":["en:japan"],
"nutriments":{"energy-kcal_100g":0,"proteins_100g":"0.1","salt_100g":0.02,"vitamin-c_100g":0.006}}`

func TestToSystemFood(t *testing.T) {
	var p Product
	if err := json.Unmarshal([]byte(sample), &p); err != nil {
		t.Fatal(err)
	}
	sf, ok := p.ToSystemFood()
	if !ok {
		t.Fatal("not converted")
	}
	if sf.Name != "緑茶" || sf.Brand != "Example" || sf.ID != "off-4901234567894" {
		t.Errorf("food = %+v", sf)
	}
	if !sf.Nutrients.Get(nutrient.VitaminC).Amount.Equal(decimal.NewFromInt(6)) {
		t.Errorf("vitamin c = %s", sf.Nutrients.Get(nutrient.VitaminC).Amount)
	}
	if !sf.Nutrients.Get(nutrient.Energy).Known() {
		t.Error("zero energy must be known")
	}
	if sf.Nutrients.Get(nutrient.Fat).Known() {
		t.Error("absent fat must be missing")
	}
	if len(sf.Portions) != 1 || !sf.Portions[0].Grams.Equal(decimal.NewFromInt(500)) {
		t.Errorf("portions = %+v", sf.Portions)
	}
}

func TestLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("User-Agent"), "Mecalo") {
			t.Error("user agent not set")
		}
		if strings.Contains(r.URL.Path, "4901234567894") {
			_, _ = w.Write([]byte(`{"status":1,"product":` + sample + `}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	p, err := c.Lookup(context.Background(), "4901234567894")
	if err != nil {
		t.Fatal(err)
	}
	if p.Revision.Name != "緑茶" || p.Revision.Source.DatasetID != DatasetID {
		t.Errorf("product = %+v", p)
	}
	if _, err := c.Lookup(context.Background(), "4900000000000"); err == nil {
		t.Error("unknown product found")
	}
}

func TestReadDump(t *testing.T) {
	line := strings.ReplaceAll(sample, "\n", "")
	dump := line + "\n" + strings.Replace(line, "en:japan", "en:france", 1) + "\nnot json\n"
	var got []app.SystemFood
	err := ReadDump(strings.NewReader(dump), "japan", func(sf app.SystemFood) error {
		got = append(got, sf)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %d products", len(got))
	}
}
