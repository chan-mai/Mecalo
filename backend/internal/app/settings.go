package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type SettingsService struct {
	d Deps
}

type SettingsInput struct {
	// ゼロ値は当日
	EffectiveFrom calendar.Date
	Profile       body.Profile
	Overrides     map[nutrient.Code]target.Override
}

func (s *SettingsService) Current(ctx context.Context, uid user.ID) (port.Settings, error) {
	return s.d.Store.Settings().At(ctx, uid, today(s.d.Clock))
}

func (s *SettingsService) History(ctx context.Context, uid user.ID) ([]port.Settings, error) {
	return s.d.Store.Settings().List(ctx, uid)
}

func (s *SettingsService) Save(ctx context.Context, uid user.ID, in SettingsInput) (port.Settings, error) {
	if in.EffectiveFrom.IsZero() {
		in.EffectiveFrom = today(s.d.Clock)
	}
	in.Profile.UserID = uid
	in.Profile.UpdatedAt = s.d.Clock.Now()
	if err := in.Profile.Validate(); err != nil {
		return port.Settings{}, invalid(err)
	}
	for code, o := range in.Overrides {
		if _, ok := nutrient.Lookup(code); !ok {
			return port.Settings{}, invalid(fmt.Errorf("unknown nutrient %q", code))
		}
		if o.Min == nil && o.Max == nil {
			delete(in.Overrides, code)
			continue
		}
		if (o.Min != nil && o.Min.IsNegative()) || (o.Max != nil && o.Max.IsNegative()) {
			return port.Settings{}, invalid(fmt.Errorf("%s: negative target", code))
		}
		if o.Min != nil && o.Max != nil && o.Min.GreaterThan(*o.Max) {
			return port.Settings{}, invalid(fmt.Errorf("%s: min exceeds max", code))
		}
	}
	st := port.Settings{UserID: uid, EffectiveFrom: in.EffectiveFrom, Profile: in.Profile, Overrides: in.Overrides}
	if err := s.d.Store.Settings().Save(ctx, st); err != nil {
		return port.Settings{}, err
	}
	return st, nil
}

// 指定日時点の設定と体重から算出, 前提不足はErrNotFound
func (s *SettingsService) TargetsAt(ctx context.Context, uid user.ID, date calendar.Date) (target.Targets, error) {
	return targetsAt(ctx, s.d.Store, s.d.Reference, uid, date)
}

func (s *SettingsService) Reference() target.Reference {
	return s.d.Reference
}

func targetsAt(ctx context.Context, r port.Repositories, ref target.Reference, uid user.ID, date calendar.Date) (target.Targets, error) {
	st, err := r.Settings().At(ctx, uid, date)
	if err != nil {
		return target.Targets{}, err
	}
	w, err := r.Weights().LatestOnOrBefore(ctx, uid, date)
	if err != nil {
		return target.Targets{}, err
	}
	ts, err := target.Build(ref, target.Person{
		Sex:          st.Profile.Sex,
		Age:          st.Profile.AgeAt(date),
		WeightKg:     w.Kg,
		Activity:     st.Profile.Activity,
		Menstruating: st.Profile.Menstruating,
	}, st.Profile.EnergyAdjustKcal, st.Overrides)
	if err != nil {
		return target.Targets{}, invalid(err)
	}
	return ts, nil
}

type Weights struct {
	d Deps
}

func (s *Weights) Save(ctx context.Context, uid user.ID, w body.Weight) (body.Weight, error) {
	w.UserID = uid
	if err := w.Validate(); err != nil {
		return body.Weight{}, invalid(err)
	}
	return w, s.d.Store.Weights().Save(ctx, w)
}

func (s *Weights) Delete(ctx context.Context, uid user.ID, d calendar.Date) error {
	return s.d.Store.Weights().Delete(ctx, uid, d)
}

// 指定日以前で最新の体重, 記録なしはfalse
func (s *Weights) LatestOnOrBefore(ctx context.Context, uid user.ID, d calendar.Date) (body.Weight, bool, error) {
	w, err := s.d.Store.Weights().LatestOnOrBefore(ctx, uid, d)
	if errors.Is(err, ErrNotFound) {
		return body.Weight{}, false, nil
	}
	return w, err == nil, err
}

func (s *Weights) Range(ctx context.Context, uid user.ID, from, to calendar.Date) ([]body.Weight, error) {
	if to.Before(from) {
		return nil, invalid(errors.New("to is before from"))
	}
	return s.d.Store.Weights().ListRange(ctx, uid, from, to)
}
