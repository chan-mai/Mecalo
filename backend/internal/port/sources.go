package port

import (
	"context"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

// 取込元での1食品
type SystemFood struct {
	ID        food.ID
	Kind      food.Kind
	Barcode   string
	Name      string
	Brand     string
	Aliases   []string
	Reference string
	Nutrients nutrient.Profile
	Basis     food.Basis
	Portions  []food.Portion
	// 取得元ページ
	URL string
}

// 外部の食品データ取得元
type FoodSource interface {
	Dataset() food.Dataset
}

// バーコードからの照会
type ProductLookup interface {
	FoodSource
	Lookup(ctx context.Context, barcode string) (SystemFood, error)
}

// 名前での外部検索
type FoodFinder interface {
	FoodSource
	Find(ctx context.Context, query string, limit int) ([]SystemFood, error)
}

// 商品一覧の定期同期
type Catalog interface {
	FoodSource
	// 全商品をsinkへ渡す, 前回から変更なしはUnchanged
	Sync(ctx context.Context, prev CatalogState, sink CatalogSink) (CatalogState, error)
}

type CatalogSink interface {
	Emit(SystemFood) error
	// 詳細の再取得を省略できる食品
	Fresh(id food.ID) bool
	// 詳細を取得せず一覧での存在のみ確認
	Seen(id food.ID) error
	// 一覧にあるが詳細を取得できなかった食品, 同期は継続
	Failed(id food.ID, err error) error
}

// 取得元ごとの同期状態
type CatalogState struct {
	// 取得元が解釈する再開・差分判定用の値
	Cursor    string
	Unchanged bool
	// 差分のみの取得, 販売終了の判定なし
	Partial bool
}

type SourceStatus struct {
	DatasetID food.DatasetID
	Cursor    string
	SyncedAt  time.Time
	// 空は成功
	Error string
	Items int
}

type SourceRepository interface {
	Status(ctx context.Context, id food.DatasetID) (SourceStatus, error)
	ListStatus(ctx context.Context) ([]SourceStatus, error)
	SaveStatus(ctx context.Context, s SourceStatus) error
	// 同期で確認した食品の記録, fetchedは詳細を取得済み
	MarkSeen(ctx context.Context, id food.DatasetID, foods []food.ID, at time.Time, fetched bool) error
	// 指定時刻以降に詳細を取得した販売中の食品
	FetchedSince(ctx context.Context, id food.DatasetID, since time.Time) ([]food.ID, error)
	// 今回確認できなかった食品の未確認回数を加算し, 上限到達分を返す
	MarkMissing(ctx context.Context, id food.DatasetID, seenBefore time.Time, limit int) ([]food.ID, error)
	CacheGet(ctx context.Context, id food.DatasetID, key string, now time.Time) ([]byte, error)
	CachePut(ctx context.Context, id food.DatasetID, key string, body []byte, expires time.Time) error
	CachePurge(ctx context.Context, now time.Time) error
}
