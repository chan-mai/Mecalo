package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/calendar"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/target"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type favorites repos

func (r favorites) Add(ctx context.Context, owner user.ID, id food.ID, at time.Time) error {
	_, err := r.q.exec(ctx,
		"INSERT INTO favorites (user_id, food_id, created_at) VALUES (?, ?, ?) ON CONFLICT (user_id, food_id) DO NOTHING",
		owner, id, r.d.timeValue(at))
	return err
}

func (r favorites) Remove(ctx context.Context, owner user.ID, id food.ID) error {
	_, err := r.q.exec(ctx, "DELETE FROM favorites WHERE user_id = ? AND food_id = ?", owner, id)
	return err
}

func (r favorites) List(ctx context.Context, owner user.ID) ([]food.ID, error) {
	rows, err := r.q.query(ctx, "SELECT food_id FROM favorites WHERE user_id = ? ORDER BY created_at DESC, food_id", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []food.ID
	for rows.Next() {
		var id food.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type searchHistory repos

const searchHistoryKeep = 50

func (r searchHistory) Record(ctx context.Context, owner user.ID, query string, at time.Time) error {
	if query == "" {
		return nil
	}
	_, err := r.q.exec(ctx, `
		INSERT INTO search_history (user_id, query, searched_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id, query) DO UPDATE SET searched_at = excluded.searched_at`,
		owner, query, r.d.timeValue(at))
	if err != nil {
		return err
	}
	_, err = r.q.exec(ctx, `
		DELETE FROM search_history WHERE user_id = ? AND query NOT IN (
			SELECT query FROM search_history WHERE user_id = ? ORDER BY searched_at DESC LIMIT ?
		)`, owner, owner, searchHistoryKeep)
	return err
}

func (r searchHistory) List(ctx context.Context, owner user.ID, limit int) ([]string, error) {
	rows, err := r.q.query(ctx,
		"SELECT query FROM search_history WHERE user_id = ? ORDER BY searched_at DESC LIMIT ?", owner, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r searchHistory) Clear(ctx context.Context, owner user.ID) error {
	_, err := r.q.exec(ctx, "DELETE FROM search_history WHERE user_id = ?", owner)
	return err
}

type settings repos

func (r settings) Save(ctx context.Context, s port.Settings) error {
	p := s.Profile
	_, err := r.q.exec(ctx, `
		INSERT INTO settings (user_id, effective_from, sex, birth_date, height_cm, activity, goal,
			energy_adjust_kcal, menstruating, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, effective_from) DO UPDATE SET
			sex = excluded.sex, birth_date = excluded.birth_date, height_cm = excluded.height_cm,
			activity = excluded.activity, goal = excluded.goal, energy_adjust_kcal = excluded.energy_adjust_kcal,
			menstruating = excluded.menstruating, updated_at = excluded.updated_at`,
		s.UserID, dateArg(s.EffectiveFrom), p.Sex, dateArg(p.BirthDate), decimalArg(p.HeightCm), p.Activity, p.Goal,
		decimalArg(p.EnergyAdjustKcal), p.Menstruating, r.d.timeValue(p.UpdatedAt))
	if err != nil {
		return err
	}
	_, err = r.q.exec(ctx, "DELETE FROM setting_overrides WHERE user_id = ? AND effective_from = ?", s.UserID, dateArg(s.EffectiveFrom))
	if err != nil {
		return err
	}
	for code, o := range s.Overrides {
		_, err := r.q.exec(ctx,
			"INSERT INTO setting_overrides (user_id, effective_from, code, min_amount, max_amount) VALUES (?, ?, ?, ?, ?)",
			s.UserID, dateArg(s.EffectiveFrom), code, nullDecimalArg(o.Min), nullDecimalArg(o.Max))
		if err != nil {
			return err
		}
	}
	return nil
}

const settingsColumns = `user_id, effective_from, sex, birth_date, height_cm, activity, goal,
	energy_adjust_kcal, menstruating, updated_at`

func (r settings) At(ctx context.Context, owner user.ID, d calendar.Date) (port.Settings, error) {
	list, err := r.list(ctx,
		"user_id = ? AND effective_from <= ? ORDER BY effective_from DESC LIMIT 1", owner, dateArg(d))
	if err != nil {
		return port.Settings{}, err
	}
	if len(list) == 0 {
		return port.Settings{}, port.ErrNotFound
	}
	return list[0], nil
}

func (r settings) List(ctx context.Context, owner user.ID) ([]port.Settings, error) {
	return r.list(ctx, "user_id = ? ORDER BY effective_from", owner)
}

func (r settings) list(ctx context.Context, where string, args ...any) ([]port.Settings, error) {
	var out []port.Settings
	err := r.q.each(ctx, "SELECT "+settingsColumns+" FROM settings WHERE "+where, args, func(rows *sql.Rows) error {
		var s port.Settings
		p := &s.Profile
		err := rows.Scan(&s.UserID, dateCol{&s.EffectiveFrom}, &p.Sex, dateCol{&p.BirthDate}, decimalCol{&p.HeightCm},
			&p.Activity, &p.Goal, decimalCol{&p.EnergyAdjustKcal}, boolCol{&p.Menstruating}, timeCol{&p.UpdatedAt})
		p.UserID = s.UserID
		s.Overrides = map[nutrient.Code]target.Override{}
		out = append(out, s)
		return err
	})
	if err != nil {
		return nil, err
	}
	for i := range out {
		if err := r.loadOverrides(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r settings) loadOverrides(ctx context.Context, s *port.Settings) error {
	rows, err := r.q.query(ctx,
		"SELECT code, min_amount, max_amount FROM setting_overrides WHERE user_id = ? AND effective_from = ?",
		s.UserID, dateArg(s.EffectiveFrom))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			code nutrient.Code
			o    target.Override
		)
		if err := rows.Scan(&code, nullDecimalCol{&o.Min}, nullDecimalCol{&o.Max}); err != nil {
			return err
		}
		s.Overrides[code] = o
	}
	return rows.Err()
}

type weights repos

func (r weights) Save(ctx context.Context, w body.Weight) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO weights (user_id, date, kg, body_fat_pct) VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, date) DO UPDATE SET kg = excluded.kg, body_fat_pct = excluded.body_fat_pct`,
		w.UserID, dateArg(w.Date), decimalArg(w.Kg), nullDecimalArg(w.BodyFatPct))
	return err
}

func (r weights) Delete(ctx context.Context, owner user.ID, d calendar.Date) error {
	return affectedOne(r.q.exec(ctx, "DELETE FROM weights WHERE user_id = ? AND date = ?", owner, dateArg(d)))
}

func (r weights) ListRange(ctx context.Context, owner user.ID, from, to calendar.Date) ([]body.Weight, error) {
	return r.list(ctx, "user_id = ? AND date >= ? AND date <= ? ORDER BY date", owner, dateArg(from), dateArg(to))
}

func (r weights) LatestOnOrBefore(ctx context.Context, owner user.ID, d calendar.Date) (body.Weight, error) {
	list, err := r.list(ctx, "user_id = ? AND date <= ? ORDER BY date DESC LIMIT 1", owner, dateArg(d))
	if err != nil {
		return body.Weight{}, err
	}
	if len(list) == 0 {
		return body.Weight{}, port.ErrNotFound
	}
	return list[0], nil
}

func (r weights) list(ctx context.Context, where string, args ...any) ([]body.Weight, error) {
	rows, err := r.q.query(ctx, "SELECT user_id, date, kg, body_fat_pct FROM weights WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []body.Weight
	for rows.Next() {
		var w body.Weight
		if err := rows.Scan(&w.UserID, dateCol{&w.Date}, decimalCol{&w.Kg}, nullDecimalCol{&w.BodyFatPct}); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
