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
	ErrUnknownMass = errors.New("food weight unknown")
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
	// 取込元から消えた時刻, 検索対象外
	DiscontinuedAt *time.Time
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

func (f Food) Discontinued() bool {
	return f.DiscontinuedAt != nil
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
	// 可食部100gあたり, Basis指定時はその1単位あたり
	Nutrients nutrient.Profile
	Basis     Basis
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
	if err := r.Basis.Validate(); err != nil {
		return err
	}
	if r.Basis.PerUnit() && !r.Basis.Grams.IsPositive() && len(r.Portions) > 0 {
		return errors.New("portions require basis grams")
	}
	seen := map[string]bool{r.Basis.Unit: r.Basis.PerUnit()}
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

// 指定数量のグラム換算, 重量不明の基準単位は0
func (r Revision) Grams(q Quantity) (decimal.Decimal, error) {
	if q.Amount.IsNegative() || q.Amount.IsZero() {
		return decimal.Zero, errors.New("quantity must be positive")
	}
	if r.Basis.PerUnit() && q.Unit == r.Basis.Unit {
		return q.Amount.Mul(r.Basis.Grams), nil
	}
	if q.Unit == UnitGram {
		if r.Basis.PerUnit() && !r.Basis.Grams.IsPositive() {
			return decimal.Zero, fmt.Errorf("%w: %s", ErrUnknownMass, r.Name)
		}
		return q.Amount, nil
	}
	for _, p := range r.Portions {
		if p.Unit == q.Unit {
			return q.Amount.Mul(p.Grams), nil
		}
	}
	return decimal.Zero, fmt.Errorf("%w: %s", ErrUnknownUnit, q.Unit)
}

// 指定数量の重量と栄養値
func (r Revision) Amounts(q Quantity) (decimal.Decimal, nutrient.Profile, error) {
	grams, err := r.Grams(q)
	if err != nil {
		return decimal.Zero, nil, err
	}
	if !r.Basis.PerUnit() {
		return grams, r.Nutrients.ForGrams(grams), nil
	}
	units := q.Amount
	if q.Unit != r.Basis.Unit {
		units = grams.Div(r.Basis.Grams)
	}
	return grams, r.Nutrients.Scale(units), nil
}

// 100gあたりの栄養値, 重量不明はfalse
func (r Revision) Per100g() (nutrient.Profile, bool) {
	if !r.Basis.PerUnit() {
		return r.Nutrients, true
	}
	if !r.Basis.Grams.IsPositive() {
		return nil, false
	}
	return r.Nutrients.Scale(hundredGrams.Div(r.Basis.Grams)), true
}

const (
	UnitGram       = "g"
	maxAliasLength = 100
)

var hundredGrams = decimal.NewFromInt(100)

// 栄養値の基準量, 単位なしは100g
type Basis struct {
	// 1個, 1杯等の単位名
	Unit string
	// 1単位の重量, 0は不明
	Grams decimal.Decimal
}

func (b Basis) PerUnit() bool {
	return b.Unit != ""
}

func (b Basis) Validate() error {
	if !b.PerUnit() {
		if !b.Grams.IsZero() {
			return errors.New("basis grams without unit")
		}
		return nil
	}
	if strings.TrimSpace(b.Unit) != b.Unit || b.Unit == UnitGram {
		return fmt.Errorf("invalid basis unit %q", b.Unit)
	}
	if b.Grams.IsNegative() {
		return errors.New("basis grams must not be negative")
	}
	return nil
}

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
	// 取得元ページ, 空は取込元全体
	URL string
	// 取得日時, ゼロ値は取込時
	RetrievedAt time.Time
}

// 取込元IDと識別子からのシステム食品ID
func SystemID(prefix, reference string) ID {
	return ID(prefix + "-" + reference)
}
