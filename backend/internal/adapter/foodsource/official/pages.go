package official

import (
	"context"
	"encoding/json"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
)

// 一覧ページ群の条件付き取得, 全ページ変更なしはunchanged
func fetchPages(ctx context.Context, f *webfetch.Client, urls []string, cursor string) (bodies [][]byte, next string, unchanged bool, err error) {
	prev := map[string]webfetch.Validators{}
	_ = json.Unmarshal([]byte(cursor), &prev)
	cur := map[string]webfetch.Validators{}
	bodies = make([][]byte, len(urls))
	var stale []int
	for i, u := range urls {
		res, err := f.Get(ctx, u, prev[u])
		if err != nil {
			return nil, cursor, false, err
		}
		cur[u] = res.Validators
		if res.NotModified {
			stale = append(stale, i)
			continue
		}
		bodies[i] = res.Body
	}
	if len(stale) == len(urls) && len(urls) > 0 {
		return nil, cursor, true, nil
	}
	// 一部のみ変更時は販売終了判定のため全ページ再取得
	for _, i := range stale {
		res, err := f.Get(ctx, urls[i], webfetch.Validators{})
		if err != nil {
			return nil, cursor, false, err
		}
		cur[urls[i]] = res.Validators
		bodies[i] = res.Body
	}
	b, err := json.Marshal(cur)
	if err != nil {
		return nil, cursor, false, err
	}
	return bodies, string(b), false, nil
}
