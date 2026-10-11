package meal

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
)

func d(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

var (
	now   = time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	today = calendar.MustParse("2026-10-12")
)

func rice() food.Revision {
	return food.Revision{
		ID:       "rev-1",
		FoodID:   "food-1",
		Number:   1,
		Name:     "こめ 精白米 うるち米 めし",
		Portions: []food.Portion{{Unit: "杯", Grams: d("150")}},
		Nutrients: nutrient.Profile{
			nutrient.Energy:     nutrient.Measured(d("156")),
			nutrient.Protein:    nutrient.Measured(d("2.5")),
			nutrient.VitaminB12: nutrient.Missing(),
		},
	}
}

func TestNewEntrySnapshot(t *testing.T) {
	rev := rice()
	e, err := NewEntry(NewEntryInput{
		ID: "e1", UserID: "u1", Date: today, Slot: SlotBreakfast,
		Quantity: food.Quantity{Amount: d("1"), Unit: "杯"}, Now: now,
	}, rev)
	if err != nil {
		t.Fatal(err)
	}

	if !e.Grams.Equal(d("150")) {
		t.Errorf("grams = %s", e.Grams)
	}
	if !e.Nutrients.Get(nutrient.Energy).Amount.Equal(d("234")) {
		t.Errorf("energy = %s", e.Nutrients.Get(nutrient.Energy).Amount)
	}
	if e.FoodRevisionID != "rev-1" || e.FoodName != rev.Name {
		t.Errorf("food reference not copied: %+v", e)
	}

	rev.Nutrients[nutrient.Energy] = nutrient.Measured(d("999"))
	if !e.Nutrients.Get(nutrient.Energy).Amount.Equal(d("234")) {
		t.Error("snapshot changed after revision mutation")
	}
}

func TestNewEntryRejectsInvalidSlot(t *testing.T) {
	_, err := NewEntry(NewEntryInput{
		Date: today, Slot: "brunch",
		Quantity: food.Quantity{Amount: d("1"), Unit: food.UnitGram}, Now: now,
	}, rice())
	if err == nil {
		t.Error("invalid slot accepted")
	}
}

func TestDuplicateKeepsSnapshot(t *testing.T) {
	e, err := NewEntry(NewEntryInput{
		ID: "e1", Date: today, Slot: SlotLunch,
		Quantity: food.Quantity{Amount: d("100"), Unit: food.UnitGram}, ClientRef: "c1", Now: now,
	}, rice())
	if err != nil {
		t.Fatal(err)
	}
	cp, err := e.Duplicate("e2", today.AddDays(1), SlotDinner, now)
	if err != nil {
		t.Fatal(err)
	}
	if cp.ID != "e2" || cp.Slot != SlotDinner || cp.ClientRef != "" {
		t.Errorf("unexpected copy: %+v", cp)
	}
	if !cp.Nutrients.Get(nutrient.Energy).Amount.Equal(d("156")) {
		t.Errorf("energy = %s", cp.Nutrients.Get(nutrient.Energy).Amount)
	}
}

func TestSummarize(t *testing.T) {
	mk := func(slot Slot, grams string, date calendar.Date) Entry {
		e, err := NewEntry(NewEntryInput{
			Date: date, Slot: slot,
			Quantity: food.Quantity{Amount: d(grams), Unit: food.UnitGram}, Now: now,
		}, rice())
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	entries := []Entry{
		mk(SlotBreakfast, "100", today),
		mk(SlotDinner, "200", today),
		mk(SlotDinner, "100", today.AddDays(-1)),
	}

	s := Summarize(today, entries)

	if s.EntryCount != 2 {
		t.Errorf("count = %d", s.EntryCount)
	}
	if !s.Total.Get(nutrient.Energy).Amount.Equal(d("468")) {
		t.Errorf("total energy = %s", s.Total.Get(nutrient.Energy).Amount)
	}
	if !s.BySlot[SlotDinner].Get(nutrient.Energy).Amount.Equal(d("312")) {
		t.Errorf("dinner energy = %s", s.BySlot[SlotDinner].Get(nutrient.Energy).Amount)
	}
	if s.Total.Get(nutrient.VitaminB12).Known() {
		t.Error("vitamin b12 should remain missing")
	}
	if !s.BySlot[SlotLunch].Get(nutrient.Energy).Known() {
		t.Error("empty slot should be known zero")
	}
}

func TestAverageOfRecordedDays(t *testing.T) {
	days := []DaySummary{
		{EntryCount: 1, Total: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("1800"))}},
		{EntryCount: 0, Total: nutrient.Sum(nil)},
		{EntryCount: 2, Total: nutrient.Profile{nutrient.Energy: nutrient.Measured(d("2200"))}},
	}
	avg, n := AverageOfRecordedDays(days)
	if n != 2 {
		t.Errorf("n = %d", n)
	}
	if !avg.Get(nutrient.Energy).Amount.Equal(d("2000")) {
		t.Errorf("avg = %s", avg.Get(nutrient.Energy).Amount)
	}
}
