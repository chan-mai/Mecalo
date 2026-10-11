// 食品, 食品リビジョン, 出典データセット
package food

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type ID string

type RevisionID string

type Kind string

const (
	// 日本食品標準成分表
	KindStandard Kind = "standard"
	// バーコード付き市販品
	KindProduct Kind = "product"
	KindCustom  Kind = "custom"
	KindRecipe  Kind = "recipe"
)

func (k Kind) Valid() bool {
	switch k {
	case KindStandard, KindProduct, KindCustom, KindRecipe:
		return true
	}
	return false
}

var (
	ErrNotFound    = errors.New("food not found")
	ErrForbidden   = errors.New("food not editable")
	ErrUnknownUnit = errors.New("unknown unit")
)

type Food struct {
	ID ID
	// 空は全利用者共通のシステム食品
	OwnerID           user.ID
	Kind              Kind
	Barcode           string
	CurrentRevisionID RevisionID
	CreatedAt         time.Time
	DeletedAt         *time.Time
}

func (f Food) IsSystem() bool {
	return f.OwnerID == ""
}

func (f Food) VisibleTo(u user.ID) bool {
	return f.IsSystem() || f.OwnerID == u
}

func (f Food) EditableBy(u user.ID) bool {
	return !f.IsSystem() && f.OwnerID == u
}

func (f Food) Deleted() bool {
	return f.DeletedAt != nil
}

// 作成後は変更しない栄養データの版
type Revision struct {
	ID     RevisionID
	FoodID ID
	// 食品内で1始まりの連番
	Number int
	Name   string
	Brand  string
	// 検索対象の別名
	Aliases []string
	// 可食部100gあたり
	Nutrients nutrient.Profile
	Portions  []Portion
	Source    Source
	CreatedAt time.Time
}

func (r Revision) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := r.Nutrients.Validate(); err != nil {
		return err
	}
	for _, a := range r.Aliases {
		if strings.TrimSpace(a) == "" || len([]rune(a)) > maxAliasLength {
			return fmt.Errorf("invalid alias %q", a)
		}
	}
	seen := map[string]bool{}
	for _, p := range r.Portions {
		if err := p.Validate(); err != nil {
			return err
		}
		if seen[p.Unit] {
			return fmt.Errorf("duplicate portion unit %q", p.Unit)
		}
		seen[p.Unit] = true
	}
	return nil
}

// 指定数量のグラム換算
func (r Revision) Grams(q Quantity) (decimal.Decimal, error) {
	if q.Amount.IsNegative() || q.Amount.IsZero() {
		return decimal.Zero, errors.New("quantity must be positive")
	}
	if q.Unit == UnitGram {
		return q.Amount, nil
	}
	for _, p := range r.Portions {
		if p.Unit == q.Unit {
			return q.Amount.Mul(p.Grams), nil
		}
	}
	return decimal.Zero, fmt.Errorf("%w: %s", ErrUnknownUnit, q.Unit)
}

const (
	UnitGram       = "g"
	maxAliasLength = 100
)

// 空白の除去と重複の除外
func NormalizeAliases(aliases []string) []string {
	out := make([]string, 0, len(aliases))
	seen := map[string]bool{}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// 個・人前などの単位とグラム数
type Portion struct {
	Unit  string
	Grams decimal.Decimal
}

func (p Portion) Validate() error {
	if strings.TrimSpace(p.Unit) == "" || p.Unit == UnitGram {
		return fmt.Errorf("invalid portion unit %q", p.Unit)
	}
	if !p.Grams.IsPositive() {
		return fmt.Errorf("portion %q grams must be positive", p.Unit)
	}
	return nil
}

type Quantity struct {
	Amount decimal.Decimal
	Unit   string
}

type Source struct {
	DatasetID DatasetID
	// 食品番号, バーコード等の取込元での識別子
	Reference string
}
