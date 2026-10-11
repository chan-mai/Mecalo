package app

import (
	"context"
	"errors"

	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type Reports struct {
	d Deps
}

type DailyReport struct {
	Summary meal.DaySummary
	// 身体情報・体重が未登録ならnil
	Targets     *target.Targets
	Evaluations []target.Evaluation
}

type RangeReport struct {
	From         calendar.Date
	To           calendar.Date
	Days         []DailyReport
	RecordedDays int
	Average      nutrient.Profile
	// 期間末日の目標に対する平均値の評価
	AverageEvaluations []target.Evaluation
}

const maxRangeDays = 366

func (s *Reports) Daily(ctx context.Context, uid user.ID, date calendar.Date) (DailyReport, error) {
	entries, err := s.d.Store.Meals().ListRange(ctx, uid, date, date)
	if err != nil {
		return DailyReport{}, err
	}
	return s.daily(ctx, uid, date, entries)
}

func (s *Reports) Range(ctx context.Context, uid user.ID, from, to calendar.Date) (RangeReport, error) {
	if to.Before(from) {
		return RangeReport{}, invalid(errors.New("to is before from"))
	}
	if from.DaysUntil(to) >= maxRangeDays {
		return RangeReport{}, invalid(errors.New("range too long"))
	}
	entries, err := s.d.Store.Meals().ListRange(ctx, uid, from, to)
	if err != nil {
		return RangeReport{}, err
	}

	out := RangeReport{From: from, To: to}
	summaries := make([]meal.DaySummary, 0, from.DaysUntil(to)+1)
	for d := from; !d.After(to); d = d.AddDays(1) {
		day, err := s.daily(ctx, uid, d, entries)
		if err != nil {
			return RangeReport{}, err
		}
		out.Days = append(out.Days, day)
		summaries = append(summaries, day.Summary)
	}
	out.Average, out.RecordedDays = meal.AverageOfRecordedDays(summaries)
	// 記録日がなければ平均の判定なし
	if last := out.Days[len(out.Days)-1]; last.Targets != nil && out.RecordedDays > 0 {
		out.AverageEvaluations = evaluate(*last.Targets, out.Average)
	}
	return out, nil
}

func (s *Reports) daily(ctx context.Context, uid user.ID, date calendar.Date, entries []meal.Entry) (DailyReport, error) {
	rep := DailyReport{Summary: meal.Summarize(date, entries)}
	ts, err := targetsAt(ctx, s.d.Store, s.d.Reference, uid, date)
	switch {
	case errors.Is(err, ErrNotFound):
		return rep, nil
	case err != nil:
		return DailyReport{}, err
	}
	rep.Targets = &ts
	// 記録のない日は摂取量0と区別できないため判定なし
	if rep.Summary.EntryCount > 0 {
		rep.Evaluations = evaluate(ts, rep.Summary.Total)
	}
	return rep, nil
}

func evaluate(ts target.Targets, total nutrient.Profile) []target.Evaluation {
	out := make([]target.Evaluation, 0, len(ts.ByCode))
	for _, def := range nutrient.Catalog() {
		t, ok := ts.ByCode[def.Code]
		if !ok {
			continue
		}
		out = append(out, target.Evaluate(t, total.Get(def.Code)))
	}
	return out
}
