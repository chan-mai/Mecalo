// 身体情報と体重記録
package body

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

type ActivityLevel string

const (
	ActivityLow      ActivityLevel = "low"
	ActivityModerate ActivityLevel = "moderate"
	ActivityHigh     ActivityLevel = "high"
)

type Goal string

const (
	GoalLose     Goal = "lose"
	GoalMaintain Goal = "maintain"
	GoalGain     Goal = "gain"
)

type Profile struct {
	UserID    user.ID
	Sex       Sex
	BirthDate calendar.Date
	HeightCm  decimal.Decimal
	Activity  ActivityLevel
	Goal      Goal
	// 目的に応じたエネルギー増減(kcal/日)
	EnergyAdjustKcal decimal.Decimal
	// 鉄の基準値選択用
	Menstruating bool
	UpdatedAt    time.Time
}

func (p Profile) Validate() error {
	switch p.Sex {
	case SexMale, SexFemale:
	default:
		return errors.New("invalid sex")
	}
	switch p.Activity {
	case ActivityLow, ActivityModerate, ActivityHigh:
	default:
		return errors.New("invalid activity level")
	}
	switch p.Goal {
	case GoalLose, GoalMaintain, GoalGain:
	default:
		return errors.New("invalid goal")
	}
	if p.BirthDate.IsZero() {
		return errors.New("birth date is required")
	}
	if !p.HeightCm.IsPositive() {
		return errors.New("height must be positive")
	}
	return nil
}

func (p Profile) AgeAt(d calendar.Date) int {
	return p.BirthDate.YearsUntil(d)
}

type Weight struct {
	UserID user.ID
	Date   calendar.Date
	Kg     decimal.Decimal
	// 体脂肪率(%), 未入力はnil
	BodyFatPct *decimal.Decimal
}

func (w Weight) Validate() error {
	if w.Date.IsZero() {
		return errors.New("date is required")
	}
	if !w.Kg.IsPositive() {
		return errors.New("weight must be positive")
	}
	if w.BodyFatPct != nil && (w.BodyFatPct.IsNegative() || w.BodyFatPct.GreaterThan(decimal.NewFromInt(100))) {
		return errors.New("body fat out of range")
	}
	return nil
}

// 指定日以前で最も新しい記録
func LatestOnOrBefore(weights []Weight, d calendar.Date) (Weight, bool) {
	var found Weight
	ok := false
	for _, w := range weights {
		if w.Date.After(d) {
			continue
		}
		if !ok || w.Date.After(found.Date) {
			found = w
			ok = true
		}
	}
	return found, ok
}
