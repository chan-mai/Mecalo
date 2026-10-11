package meal

import (
	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

type DaySummary struct {
	Date       calendar.Date
	EntryCount int
	Total      nutrient.Profile
	BySlot     map[Slot]nutrient.Profile
}

// スナップショットのみから日次集計
func Summarize(date calendar.Date, entries []Entry) DaySummary {
	all := make([]nutrient.Part, 0, len(entries))
	bySlot := map[Slot][]nutrient.Part{}
	count := 0
	for _, e := range entries {
		if !e.Date.Equal(date) {
			continue
		}
		all = append(all, e.part())
		bySlot[e.Slot] = append(bySlot[e.Slot], e.part())
		count++
	}

	s := DaySummary{
		Date:       date,
		EntryCount: count,
		Total:      nutrient.Sum(all),
		BySlot:     make(map[Slot]nutrient.Profile, len(Slots())),
	}
	for _, slot := range Slots() {
		s.BySlot[slot] = nutrient.Sum(bySlot[slot])
	}
	return s
}

// 記録のある日のみの1日平均
func AverageOfRecordedDays(days []DaySummary) (nutrient.Profile, int) {
	parts := make([]nutrient.Part, 0, len(days))
	for _, day := range days {
		if day.EntryCount == 0 {
			continue
		}
		parts = append(parts, nutrient.Part{Grams: decimal.NewFromInt(1), Values: day.Total})
	}
	if len(parts) == 0 {
		return nutrient.Sum(nil), 0
	}
	n := decimal.NewFromInt(int64(len(parts)))
	return nutrient.Sum(parts).Scale(decimal.NewFromInt(1).Div(n)), len(parts)
}
