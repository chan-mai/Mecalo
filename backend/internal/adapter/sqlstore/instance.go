package sqlstore

import (
	"context"
	"database/sql"
	"time"
)

type instance repos

func (r instance) List(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	err := r.q.each(ctx, "SELECT key, value FROM instance_settings", nil, func(rows *sql.Rows) error {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		out[k] = v
		return nil
	})
	return out, err
}

func (r instance) Put(ctx context.Context, values map[string]string, at time.Time) error {
	now := r.d.timeValue(at)
	for k, v := range values {
		_, err := r.q.exec(ctx, `
			INSERT INTO instance_settings (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			k, v, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r instance) Delete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		_, err := r.q.exec(ctx, "DELETE FROM instance_settings")
		return err
	}
	for _, k := range keys {
		if _, err := r.q.exec(ctx, "DELETE FROM instance_settings WHERE key = ?", k); err != nil {
			return err
		}
	}
	return nil
}
