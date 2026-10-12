// app層が依存する外部境界インタフェース
package port

import (
	"context"
	"errors"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/recipe"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store interface {
	Repositories
	// fnがエラーを返した場合はロールバック
	InTx(ctx context.Context, fn func(Repositories) error) error
	Close() error
}

type Repositories interface {
	Users() UserRepository
	Sessions() SessionRepository
	Datasets() DatasetRepository
	Foods() FoodRepository
	Recipes() RecipeRepository
	Meals() MealRepository
	Favorites() FavoriteRepository
	SearchHistory() SearchHistoryRepository
	QueryLinks() QueryLinkRepository
	Settings() SettingsRepository
	Weights() WeightRepository
	Sources() SourceRepository
}

type Credential struct {
	UserID user.ID
	// "local" またはOIDC issuer
	Provider string
	// localはログイン名, OIDCはsub
	Subject      string
	PasswordHash string
}

type UserRepository interface {
	Create(ctx context.Context, u user.User, cred Credential) error
	Get(ctx context.Context, id user.ID) (user.User, error)
	FindByCredential(ctx context.Context, provider, subject string) (user.User, Credential, error)
	LocalCredential(ctx context.Context, id user.ID) (Credential, error)
	List(ctx context.Context) ([]user.User, error)
	Count(ctx context.Context) (int, error)
	UpdatePassword(ctx context.Context, id user.ID, hash string) error
	Delete(ctx context.Context, id user.ID) error
}

type Session struct {
	// トークンのハッシュ
	ID        string
	UserID    user.ID
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, id string) (Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context, now time.Time) error
}

type DatasetRepository interface {
	Upsert(ctx context.Context, d food.Dataset) error
	Get(ctx context.Context, id food.DatasetID) (food.Dataset, error)
	List(ctx context.Context) ([]food.Dataset, error)
}

type FoodSearchQuery struct {
	UserID user.ID
	// 語ごとの検索キー候補(表記・読み), 全語に一致するものが対象
	Terms  [][]string
	Kinds  []food.Kind
	Limit  int
	Offset int
}

// 検索結果1件
type FoodHit struct {
	Food     food.Food
	Revision food.Revision
}

// 2文字の組の重なりでの検索
type SimilarQuery struct {
	UserID user.ID
	Grams  []string
	// 共有する組の最小数
	MinShared int
	Kinds     []food.Kind
	Limit     int
}

type SimilarHit struct {
	FoodHit
	// 検索語と共有する組の数
	Shared int
	// 候補の組の総数
	Grams int
}

type FoodRepository interface {
	// foodと最初のリビジョンを作成
	Create(ctx context.Context, f food.Food, rev food.Revision) error
	// リビジョン追加と現行リビジョンの切替
	AddRevision(ctx context.Context, rev food.Revision) error
	// 利用者から見えない食品はErrNotFound
	Get(ctx context.Context, viewer user.ID, id food.ID) (food.Food, food.Revision, error)
	GetRevision(ctx context.Context, id food.RevisionID) (food.Revision, error)
	ListRevisions(ctx context.Context, viewer user.ID, id food.ID) ([]food.Revision, error)
	FindByBarcode(ctx context.Context, viewer user.ID, barcode string) (food.Food, food.Revision, error)
	FindBySource(ctx context.Context, dataset food.DatasetID, reference string) (food.Food, food.Revision, error)
	Search(ctx context.Context, q FoodSearchQuery) ([]FoodHit, error)
	SearchSimilar(ctx context.Context, q SimilarQuery) ([]SimilarHit, error)
	ListOwned(ctx context.Context, owner user.ID) ([]FoodHit, error)
	SoftDelete(ctx context.Context, id food.ID, at time.Time) error
	// nilで販売終了を解除
	SetDiscontinued(ctx context.Context, id food.ID, at *time.Time) error
}

type RecipeRepository interface {
	// recipe種別リビジョンの材料構成を保存
	Save(ctx context.Context, rev food.RevisionID, res recipe.Result) error
	Get(ctx context.Context, rev food.RevisionID) (recipe.Result, error)
}

type MealRepository interface {
	Create(ctx context.Context, e meal.Entry) error
	Update(ctx context.Context, e meal.Entry) error
	Get(ctx context.Context, owner user.ID, id meal.ID) (meal.Entry, error)
	FindByClientRef(ctx context.Context, owner user.ID, ref string) (meal.Entry, error)
	Delete(ctx context.Context, owner user.ID, id meal.ID) error
	// from, toを含む
	ListRange(ctx context.Context, owner user.ID, from, to calendar.Date) ([]meal.Entry, error)
	// 食品ごとの直近記録, 新しい順
	Recent(ctx context.Context, owner user.ID, limit int) ([]meal.Entry, error)
	// 記録回数の多い食品
	Frequent(ctx context.Context, owner user.ID, since calendar.Date, limit int) ([]FoodUsage, error)
}

type FoodUsage struct {
	FoodID food.ID
	Count  int
	// 最後に記録した数量
	LastQuantity food.Quantity
}

type FavoriteRepository interface {
	Add(ctx context.Context, owner user.ID, id food.ID, at time.Time) error
	Remove(ctx context.Context, owner user.ID, id food.ID) error
	List(ctx context.Context, owner user.ID) ([]food.ID, error)
}

type SearchHistoryRepository interface {
	Record(ctx context.Context, owner user.ID, query string, at time.Time) error
	List(ctx context.Context, owner user.ID, limit int) ([]string, error)
	Clear(ctx context.Context, owner user.ID) error
}

// 検索語と、その検索から利用者が選んだ食品の対応
type QueryLinkRepository interface {
	Link(ctx context.Context, owner user.ID, queryKey string, id food.ID, at time.Time) error
	// 選択回数の多い順
	Linked(ctx context.Context, owner user.ID, queryKey string, limit int) ([]food.ID, error)
}

// 適用開始日付きの身体情報と目標設定
type Settings struct {
	UserID        user.ID
	EffectiveFrom calendar.Date
	Profile       body.Profile
	Overrides     map[nutrient.Code]target.Override
}

type SettingsRepository interface {
	// 同一適用開始日は上書き
	Save(ctx context.Context, s Settings) error
	// 指定日時点で有効な設定
	At(ctx context.Context, owner user.ID, d calendar.Date) (Settings, error)
	List(ctx context.Context, owner user.ID) ([]Settings, error)
}

type WeightRepository interface {
	// 同一日は上書き
	Save(ctx context.Context, w body.Weight) error
	Delete(ctx context.Context, owner user.ID, d calendar.Date) error
	ListRange(ctx context.Context, owner user.ID, from, to calendar.Date) ([]body.Weight, error)
	LatestOnOrBefore(ctx context.Context, owner user.ID, d calendar.Date) (body.Weight, error)
}
