// ユースケース層, IOはport経由のみ
package app

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

var (
	ErrNotFound     = port.ErrNotFound
	ErrConflict     = port.ErrConflict
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalid      = errors.New("invalid input")
	ErrDisabled     = errors.New("feature disabled")
)

// 入力検証エラーとして包む
func Invalid(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalid, err)
}

func invalid(err error) error {
	return Invalid(err)
}

type Deps struct {
	Store     port.Store
	Clock     port.Clock
	IDs       port.IDGenerator
	Reference target.Reference
	// nilは読みでの照合なし
	Reader port.TextReader
	// nilは外部照会無効
	Lookup port.ProductLookup
	// 利用者操作時のみ使う外部検索
	Finders []port.FoodFinder
	// 定期同期する商品一覧
	Catalogs []port.Catalog
	// 初回の管理者作成に必要なトークン, 空は不要
	SetupToken string
	// nilはClient ID Metadata Document非対応
	ClientMetadata port.ClientMetadataFetcher
	// 同期の経過の通知先, nilは通知なし
	OnSync func(SyncEvent)
	Hasher port.PasswordHasher
}

// 全ユースケース
type App struct {
	Accounts *Accounts
	Foods    *Foods
	Meals    *Meals
	Recipes  *Recipes
	Reports  *Reports
	Settings *SettingsService
	Weights  *Weights
	Exports  *Exports
	Imports  *Imports
	Sources  *Sources
	Tokens   *Tokens
	OAuth    *OAuth
	// 管理者が変更する設定, 未設定はnil
	Instance *Instance

	live *live
}

type Options struct {
	RegistrationOpen  bool
	OIDCAutoProvision bool
	MCPEnabled        bool
	// 末尾の/なし, 空はOAuth無効
	PublicURL string
}

func New(d Deps, opt Options) *App {
	imports := &Imports{d: d}
	l := newLive(d, opt)
	oauth := &OAuth{d: d, live: l}
	return &App{
		Accounts: &Accounts{d: d, live: l},
		Foods:    &Foods{d: d, imports: imports, live: l},
		Meals:    &Meals{d: d},
		Recipes:  &Recipes{d: d},
		Reports:  &Reports{d: d},
		Settings: &SettingsService{d: d},
		Weights:  &Weights{d: d},
		Exports:  &Exports{d: d},
		Imports:  imports,
		Sources:  &Sources{d: d, imports: imports, live: l},
		Tokens:   &Tokens{d: d, live: l, oauth: oauth},
		OAuth:    oauth,
		live:     l,
	}
}

var (
	one          = decimal.NewFromInt(1)
	hundredGrams = decimal.NewFromInt(100)
)

// インスタンスのタイムゾーンでの当日
func (a *App) Today() calendar.Date {
	return today(a.Settings.d.Clock)
}

func today(c port.Clock) calendar.Date {
	now := c.Now()
	return calendar.In(now, now.Location())
}
