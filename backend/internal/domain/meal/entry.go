// 食事記録と日次集計
package meal

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type ID string

type Slot string

const (
	SlotBreakfast Slot = "breakfast"
	SlotLunch     Slot = "lunch"
	SlotDinner    Slot = "dinner"
	SlotSnack     Slot = "snack"
)

func Slots() []Slot {
	return []Slot{SlotBreakfast, SlotLunch, SlotDinner, SlotSnack}
}

func (s Slot) Valid() bool {
	switch s {
	case SlotBreakfast, SlotLunch, SlotDinner, SlotSnack:
		return true
	}
	return false
}

var ErrNotFound = errors.New("meal entry not found")

// 記録時点の食品情報と栄養値を複製して保持
type Entry struct {
	ID     ID
	UserID user.ID
	Date   calendar.Date
	Slot   Slot

	FoodID         food.ID
	FoodRevisionID food.RevisionID
	FoodName       string
	Brand          string
	Source         food.Source

	Quantity food.Quantity
	Grams    decimal.Decimal
	// Grams分の栄養値
	Nutrients nutrient.Profile

	// クライアント生成の冪等キー
	ClientRef string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewEntryInput struct {
	ID        ID
	UserID    user.ID
	Date      calendar.Date
	Slot      Slot
	Quantity  food.Quantity
	ClientRef string
	Now       time.Time
}

func NewEntry(in NewEntryInput, rev food.Revision) (Entry, error) {
	if !in.Slot.Valid() {
		return Entry{}, fmt.Errorf("invalid slot %q", in.Slot)
	}
	if in.Date.IsZero() {
		return Entry{}, errors.New("date is required")
	}
	e := Entry{
		ID:        in.ID,
		UserID:    in.UserID,
		Date:      in.Date,
		Slot:      in.Slot,
		ClientRef: in.ClientRef,
		CreatedAt: in.Now,
	}
	return e.Resolve(rev, in.Quantity, in.Now)
}

// 指定リビジョンと数量でスナップショットを再生成
func (e Entry) Resolve(rev food.Revision, q food.Quantity, now time.Time) (Entry, error) {
	grams, err := rev.Grams(q)
	if err != nil {
		return Entry{}, err
	}
	e.FoodID = rev.FoodID
	e.FoodRevisionID = rev.ID
	e.FoodName = rev.Name
	e.Brand = rev.Brand
	e.Source = rev.Source
	e.Quantity = q
	e.Grams = grams
	e.Nutrients = rev.Nutrients.ForGrams(grams)
	e.UpdatedAt = now
	return e, nil
}

// スナップショットをそのまま別の日・区分へ複製
func (e Entry) Duplicate(id ID, date calendar.Date, slot Slot, now time.Time) (Entry, error) {
	if !slot.Valid() {
		return Entry{}, fmt.Errorf("invalid slot %q", slot)
	}
	e.ID = id
	e.Date = date
	e.Slot = slot
	e.ClientRef = ""
	e.CreatedAt = now
	e.UpdatedAt = now
	return e, nil
}

func (e Entry) part() nutrient.Part {
	return nutrient.Part{Grams: e.Grams, Values: e.Nutrients}
}
