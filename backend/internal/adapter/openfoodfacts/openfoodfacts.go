// Open Food Factsの商品データ変換と照会
package openfoodfacts

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	DatasetID   = food.DatasetID("openfoodfacts")
	DefaultBase = "https://world.openfoodfacts.org"
	servingUnit = "1食分"
)

var barcodePattern = regexp.MustCompile(`^\d{8,14}$`)

func Dataset() food.Dataset {
	return food.Dataset{
		ID:          DatasetID,
		Kind:        food.DatasetOpenFoodFacts,
		Name:        "Open Food Facts",
		License:     "ODbL 1.0 / DbCL 1.0",
		URL:         "https://world.openfoodfacts.org",
		Attribution: "Contains data from Open Food Facts, available under the Open Database License.",
	}
}

// _100gの値はすべてグラム単位
var nutrimentKeys = map[nutrient.Code]string{
	nutrient.Energy:          "energy-kcal_100g",
	nutrient.Protein:         "proteins_100g",
	nutrient.Fat:             "fat_100g",
	nutrient.Carbohydrate:    "carbohydrates_100g",
	nutrient.Fiber:           "fiber_100g",
	nutrient.SaltEquivalent:  "salt_100g",
	nutrient.Cholesterol:     "cholesterol_100g",
	nutrient.Sodium:          "sodium_100g",
	nutrient.Potassium:       "potassium_100g",
	nutrient.Calcium:         "calcium_100g",
	nutrient.Magnesium:       "magnesium_100g",
	nutrient.Phosphorus:      "phosphorus_100g",
	nutrient.Iron:            "iron_100g",
	nutrient.Zinc:            "zinc_100g",
	nutrient.Copper:          "copper_100g",
	nutrient.Manganese:       "manganese_100g",
	nutrient.Iodine:          "iodine_100g",
	nutrient.Selenium:        "selenium_100g",
	nutrient.Chromium:        "chromium_100g",
	nutrient.Molybdenum:      "molybdenum_100g",
	nutrient.VitaminA:        "vitamin-a_100g",
	nutrient.VitaminD:        "vitamin-d_100g",
	nutrient.VitaminE:        "vitamin-e_100g",
	nutrient.VitaminK:        "vitamin-k_100g",
	nutrient.VitaminB1:       "vitamin-b1_100g",
	nutrient.VitaminB2:       "vitamin-b2_100g",
	nutrient.Niacin:          "vitamin-pp_100g",
	nutrient.VitaminB6:       "vitamin-b6_100g",
	nutrient.VitaminB12:      "vitamin-b12_100g",
	nutrient.Folate:          "vitamin-b9_100g",
	nutrient.PantothenicAcid: "pantothenic-acid_100g",
	nutrient.Biotin:          "biotin_100g",
	nutrient.VitaminC:        "vitamin-c_100g",
}

// 100gあたりの上限, キーはエネルギーか否か
var maxPer100g = map[bool]decimal.Decimal{
	true:  decimal.NewFromInt(900),
	false: decimal.NewFromInt(100),
}

var gramFactor = map[nutrient.Unit]decimal.Decimal{
	nutrient.UnitKcal:      decimal.NewFromInt(1),
	nutrient.UnitGram:      decimal.NewFromInt(1),
	nutrient.UnitMilligram: decimal.NewFromInt(1_000),
	nutrient.UnitMicrogram: decimal.NewFromInt(1_000_000),
}

type Product struct {
	Code          string                     `json:"code"`
	ProductName   string                     `json:"product_name"`
	ProductNameJA string                     `json:"product_name_ja"`
	Brands        string                     `json:"brands"`
	ServingQty    json.RawMessage            `json:"serving_quantity"`
	CountriesTags []string                   `json:"countries_tags"`
	Nutriments    map[string]json.RawMessage `json:"nutriments"`
}

// 栄養値なし・商品名なしはfalse
func (p Product) ToSystemFood() (app.SystemFood, bool) {
	name := strings.TrimSpace(p.ProductNameJA)
	if name == "" {
		name = strings.TrimSpace(p.ProductName)
	}
	code, ok := food.NormalizeBarcode(p.Code)
	if name == "" || !ok {
		return app.SystemFood{}, false
	}
	sf := app.SystemFood{
		ID:        food.SystemID("off", code),
		Kind:      food.KindProduct,
		Barcode:   code,
		Name:      name,
		Brand:     firstBrand(p.Brands),
		Reference: code,
		Nutrients: nutrient.Profile{},
		URL:       DefaultBase + "/product/" + code,
	}
	for code, key := range nutrimentKeys {
		v, ok := number(p.Nutriments[key])
		// 100gあたりで成立しない値は入力誤りとして除外
		if !ok || v.IsNegative() || v.GreaterThan(maxPer100g[code == nutrient.Energy]) {
			continue
		}
		def, _ := nutrient.Lookup(code)
		sf.Nutrients[code] = nutrient.Measured(v.Mul(gramFactor[def.Unit]))
	}
	if len(sf.Nutrients) == 0 {
		return app.SystemFood{}, false
	}
	if g, ok := number(p.ServingQty); ok && g.IsPositive() {
		sf.Portions = []food.Portion{{Unit: servingUnit, Grams: g}}
	}
	return sf, true
}

func firstBrand(brands string) string {
	b, _, _ := strings.Cut(brands, ",")
	return strings.TrimSpace(b)
}

// 数値と文字列の両表記に対応
func number(raw json.RawMessage) (decimal.Decimal, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return decimal.Zero, false
	}
	s := strings.Trim(string(raw), `"`)
	if s == "" {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(s)
	return d, err == nil
}

// 外部照会, 利用者のスキャン1回につき1リクエスト
type Client struct {
	Base      string
	UserAgent string
	HTTP      *http.Client
}

var _ port.ProductLookup = (*Client)(nil)

func NewClient(base, userAgent string) *Client {
	if base == "" {
		base = DefaultBase
	}
	return &Client{Base: strings.TrimRight(base, "/"), UserAgent: userAgent, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Dataset() food.Dataset {
	return Dataset()
}

func (c *Client) Lookup(ctx context.Context, barcode string) (app.SystemFood, error) {
	if !barcodePattern.MatchString(barcode) {
		return app.SystemFood{}, port.ErrNotFound
	}
	fields := "code,product_name,product_name_ja,brands,serving_quantity,nutriments"
	u := fmt.Sprintf("%s/api/v2/product/%s.json?fields=%s", c.Base, url.PathEscape(barcode), fields)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return app.SystemFood{}, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return app.SystemFood{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return app.SystemFood{}, port.ErrNotFound
	}
	if res.StatusCode != http.StatusOK {
		return app.SystemFood{}, fmt.Errorf("open food facts: status %d", res.StatusCode)
	}
	var body struct {
		Status  int     `json:"status"`
		Product Product `json:"product"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&body); err != nil {
		return app.SystemFood{}, err
	}
	if body.Status != 1 {
		return app.SystemFood{}, port.ErrNotFound
	}
	if body.Product.Code == "" {
		body.Product.Code = barcode
	}
	sf, ok := body.Product.ToSystemFood()
	if !ok {
		return app.SystemFood{}, port.ErrNotFound
	}
	return sf, nil
}

// JSONLダンプ(gzip可)から指定国の商品を抽出
func ReadDump(r io.Reader, country string, fn func(app.SystemFood) error) error {
	br := bufio.NewReader(r)
	if head, err := br.Peek(2); err == nil && head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return err
		}
		defer gz.Close()
		br = bufio.NewReader(gz)
	}
	sc := bufio.NewScanner(br)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	tag := "en:" + strings.ToLower(country)
	for sc.Scan() {
		line := sc.Bytes()
		if !strings.Contains(string(line), tag) {
			continue
		}
		var p Product
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		if !slices.Contains(p.CountriesTags, tag) {
			continue
		}
		sf, ok := p.ToSystemFood()
		if !ok {
			continue
		}
		if err := fn(sf); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
