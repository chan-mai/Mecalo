// タベカル(外食・中食チェーンの公式公表値API)の検索
package tabecal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	DatasetID   = food.DatasetID("tabecal")
	DefaultBase = "https://tabecal.com/api/v1"
	// キーなしの1回あたり上限
	anonymousLimit = 50
	servingUnit    = "1食"
)

func Dataset() food.Dataset {
	return food.Dataset{
		ID:      DatasetID,
		Kind:    food.DatasetTabecal,
		Name:    "タベカル",
		License: "Tabecal API terms (no redistribution)",
		URL:     "https://tabecal.com/api/",
		// 利用規約で指定された表記
		Attribution: "データ: タベカル（tabecal.com）／各社公式公表値",
	}
}

type Client struct {
	Base string
	// 空はキーなし(Freeプラン相当)
	Key       string
	UserAgent string
	HTTP      *http.Client

	mu        sync.Mutex
	chainList []chain
	chainsAt  time.Time
}

var _ port.FoodFinder = (*Client)(nil)

func New(base, key, userAgent string) *Client {
	if base == "" {
		base = DefaultBase
	}
	return &Client{Base: strings.TrimRight(base, "/"), Key: key, UserAgent: userAgent, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Dataset() food.Dataset {
	return Dataset()
}

type item struct {
	ItemKey   string   `json:"item_key"`
	Chain     string   `json:"chain"`
	Name      string   `json:"name"`
	Size      string   `json:"size"`
	Kcal      *float64 `json:"kcal"`
	Protein   *float64 `json:"protein_g"`
	Fat       *float64 `json:"fat_g"`
	Carb      *float64 `json:"carb_g"`
	Fiber     *float64 `json:"fiber_g"`
	Sugar     *float64 `json:"sugar_g"`
	Salt      *float64 `json:"salt_g"`
	Sodium    *float64 `json:"sodium_mg"`
	Weight    *float64 `json:"weight_g"`
	ValidFrom string   `json:"valid_from"`
}

// 部分一致の検索に加え, チェーン名と類似する語はチェーン内を検索
func (c *Client) Find(ctx context.Context, query string, limit int) ([]app.SystemFood, error) {
	if c.Key == "" {
		limit = min(limit, anonymousLimit)
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	items, err := c.items(ctx, q)
	if err != nil {
		return nil, err
	}
	if slug, rest, ok := c.matchChain(ctx, query); ok {
		q := url.Values{}
		q.Set("chain", slug)
		q.Set("limit", strconv.Itoa(limit))
		if rest != "" {
			q.Set("q", rest)
		}
		more, err := c.items(ctx, q)
		if err != nil {
			return nil, err
		}
		items = append(items, more...)
	}
	seen := map[string]bool{}
	out := make([]app.SystemFood, 0, len(items))
	for _, it := range items {
		if seen[it.ItemKey] {
			continue
		}
		seen[it.ItemKey] = true
		if sf, ok := it.systemFood(); ok {
			out = append(out, sf)
		}
	}
	rankBySimilarity(out, query)
	return out, nil
}

func (c *Client) items(ctx context.Context, q url.Values) ([]item, error) {
	var body struct {
		Data []item `json:"data"`
	}
	if err := c.get(ctx, "/items?"+q.Encode(), &body); err != nil {
		return nil, err
	}
	return body.Data, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Key != "" {
		req.Header.Set("X-Api-Key", c.Key)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("tabecal: status %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out)
}

type chain struct {
	Slug string `json:"slug"`
	Name string `json:"name_ja"`
}

// チェーン一覧の再取得間隔
const chainsTTL = 24 * time.Hour

func (c *Client) chains(ctx context.Context) ([]chain, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.chainList != nil && time.Since(c.chainsAt) < chainsTTL {
		return c.chainList, nil
	}
	var body struct {
		Data []chain `json:"data"`
	}
	if err := c.get(ctx, "/chains", &body); err != nil {
		return nil, err
	}
	c.chainList, c.chainsAt = body.Data, time.Now()
	return c.chainList, nil
}

// 語とチェーン名の組がともにこの割合以上重なればそのチェーンとみなす
const chainOverlap = 0.75

// チェーン名と類似する語と, それ以外の語, チェーン一覧の取得失敗は該当なし
func (c *Client) matchChain(ctx context.Context, query string) (slug, rest string, ok bool) {
	list, err := c.chains(ctx)
	if err != nil {
		return "", "", false
	}
	words := food.SearchTerms(query)
	best, bestWord, bestScore := "", -1, 0.0
	for i, w := range words {
		wg := food.Bigrams(w)
		for _, ch := range list {
			cg := food.Bigrams(food.SearchKey(ch.Name))
			shared := countShared(wg, cg)
			if len(wg) == 0 || len(cg) == 0 {
				if w == food.SearchKey(ch.Name) {
					shared, wg, cg = 1, []string{w}, []string{w}
				} else {
					continue
				}
			}
			byWord, dice := food.Similarity(shared, len(wg), len(cg))
			byChain := float64(shared) / float64(len(cg))
			if byWord >= chainOverlap && byChain >= chainOverlap && dice > bestScore {
				best, bestWord, bestScore = ch.Slug, i, dice
			}
		}
	}
	if best == "" {
		return "", "", false
	}
	others := append(append([]string{}, words[:bestWord]...), words[bestWord+1:]...)
	return best, strings.Join(others, " "), true
}

func countShared(a, b []string) int {
	set := make(map[string]bool, len(b))
	for _, g := range b {
		set[g] = true
	}
	n := 0
	for _, g := range a {
		if set[g] {
			n++
		}
	}
	return n
}

// 検索語との組の重なりが大きい順, 同値は元の順
func rankBySimilarity(foods []app.SystemFood, query string) {
	qg := food.Bigrams(strings.Join(food.SearchTerms(query), ""))
	score := make(map[string]float64, len(foods))
	for _, f := range foods {
		g := food.BigramsOf(food.SearchKey(f.Name), food.SearchKey(f.Brand))
		_, score[f.Reference] = food.Similarity(countShared(qg, g), len(qg), len(g))
	}
	sort.SliceStable(foods, func(i, j int) bool { return score[foods[i].Reference] > score[foods[j].Reference] })
}

func (it item) systemFood() (app.SystemFood, bool) {
	name := strings.TrimSpace(it.Name)
	if it.ItemKey == "" || name == "" {
		return app.SystemFood{}, false
	}
	if size := strings.TrimSpace(it.Size); size != "" && !strings.Contains(name, size) {
		name += " " + size
	}
	var v catalog.Values
	set := func(code nutrient.Code, x *float64, unit string) {
		if x != nil && *x >= 0 {
			v.SetNumber(code, decimal.NewFromFloat(*x), unit)
		}
	}
	set(nutrient.Energy, it.Kcal, "kcal")
	set(nutrient.Protein, it.Protein, "g")
	set(nutrient.Fat, it.Fat, "g")
	set(nutrient.Carbohydrate, it.Carb, "g")
	set(nutrient.Fiber, it.Fiber, "g")
	set(nutrient.SaltEquivalent, it.Salt, "g")
	set(nutrient.Sodium, it.Sodium, "mg")
	if it.Sugar != nil {
		v.SetSugar(decimal.NewFromFloat(*it.Sugar))
	}
	if v.Len() == 0 {
		return app.SystemFood{}, false
	}
	basis := food.Basis{Unit: servingUnit}
	if it.Weight != nil && *it.Weight > 0 {
		basis.Grams = decimal.NewFromFloat(*it.Weight)
	}
	return app.SystemFood{
		ID:        food.SystemID("tabecal", it.ItemKey),
		Kind:      food.KindProduct,
		Name:      name,
		Brand:     strings.TrimSpace(it.Chain),
		Reference: it.ItemKey,
		Nutrients: v.Profile(),
		Basis:     basis,
	}, true
}
