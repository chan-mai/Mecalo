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
	Barcode port.BarcodeLookup
	Hasher  port.PasswordHasher
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
}

type Options struct {
	RegistrationOpen  bool
	OIDCAutoProvision bool
}

func New(d Deps, opt Options) *App {
	return &App{
		Accounts: &Accounts{d: d, opt: opt},
		Foods:    &Foods{d: d},
		Meals:    &Meals{d: d},
		Recipes:  &Recipes{d: d},
		Reports:  &Reports{d: d},
		Settings: &SettingsService{d: d},
		Weights:  &Weights{d: d},
		Exports:  &Exports{d: d},
		Imports:  &Imports{d: d},
	}
}

var (
	one          = decimal.NewFromInt(1)
	hundredGrams = decimal.NewFromInt(100)
)

func today(c port.Clock) calendar.Date {
	now := c.Now()
	return calendar.In(now, now.Location())
}
