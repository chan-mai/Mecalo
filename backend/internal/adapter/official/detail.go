package official

import (
	"context"
	"errors"

	"github.com/chan-mai/mecalo/backend/internal/adapter/catalog"
	"github.com/chan-mai/mecalo/backend/internal/adapter/webfetch"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 商品ページの取得と解析, 取得済みは確認のみ, 404は読み飛ばし
func fetchDetail(ctx context.Context, f *webfetch.Client, sink port.CatalogSink, ids []food.ID, url string, parse func(catalog.Node) []app.SystemFood) error {
	if skipped, err := seenFresh(sink, ids); skipped || err != nil {
		return err
	}
	res, err := f.Get(ctx, url, webfetch.Validators{})
	if errors.Is(err, webfetch.ErrNotFound) {
		return nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		return failAll(sink, ids, err)
	}
	doc, err := catalog.ParseHTML(res.Body)
	if err != nil {
		return err
	}
	for _, sf := range parse(doc) {
		if err := sink.Emit(sf); err != nil {
			return err
		}
	}
	return nil
}

// 候補IDのいずれかが詳細取得済みなら確認のみ
func seenFresh(sink port.CatalogSink, ids []food.ID) (bool, error) {
	skipped := false
	for _, id := range ids {
		if sink.Fresh(id) {
			skipped = true
			if err := sink.Seen(id); err != nil {
				return true, err
			}
		}
	}
	return skipped, nil
}

// 前回の分け方が不明なため候補ID全てを失敗として通知
func failAll(sink port.CatalogSink, ids []food.ID, err error) error {
	for _, id := range ids {
		if e := sink.Failed(id, err); e != nil {
			return e
		}
	}
	return nil
}

// 1件だけの解析結果を一覧へ
func single(sf app.SystemFood, ok bool) []app.SystemFood {
	if !ok {
		return nil
	}
	return []app.SystemFood{sf}
}
