package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type sources repos

const sourceStatusColumns = "dataset_id, cursor, synced_at, error, items"

func scanSourceStatus(row interface{ Scan(...any) error }) (port.SourceStatus, error) {
	var s port.SourceStatus
	err := row.Scan(&s.DatasetID, &s.Cursor, timeCol{&s.SyncedAt}, &s.Error, &s.Items)
	return s, notFound(err)
}

func (r sources) Status(ctx context.Context, id food.DatasetID) (port.SourceStatus, error) {
	return scanSourceStatus(r.q.queryRow(ctx, "SELECT "+sourceStatusColumns+" FROM source_status WHERE dataset_id = ?", id))
}

func (r sources) ListStatus(ctx context.Context) ([]port.SourceStatus, error) {
	var out []port.SourceStatus
	err := r.q.each(ctx, "SELECT "+sourceStatusColumns+" FROM source_status ORDER BY dataset_id", nil, func(rows *sql.Rows) error {
		s, err := scanSourceStatus(rows)
		out = append(out, s)
		return err
	})
	return out, err
}

func (r sources) SaveStatus(ctx context.Context, s port.SourceStatus) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO source_status (dataset_id, cursor, synced_at, error, items) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (dataset_id) DO UPDATE SET
			cursor = excluded.cursor, synced_at = excluded.synced_at, error = excluded.error, items = excluded.items`,
		s.DatasetID, s.Cursor, r.d.timeValue(s.SyncedAt), s.Error, s.Items)
	return err
}

func (r sources) MarkSeen(ctx context.Context, id food.DatasetID, foods []food.ID, at time.Time, fetched bool) error {
	update := "seen_at = excluded.seen_at, misses = 0"
	if fetched {
		update += ", fetched_at = excluded.fetched_at"
	}
	for _, fid := range foods {
		// 未登録の食品は対象外
		_, err := r.q.exec(ctx, `
			INSERT INTO source_items (dataset_id, food_id, seen_at, fetched_at, misses)
			SELECT ?, id, ?, ?, 0 FROM foods WHERE id = ?
			ON CONFLICT (dataset_id, food_id) DO UPDATE SET `+update,
			id, r.d.timeValue(at), r.d.timeValue(at), fid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r sources) FetchedSince(ctx context.Context, id food.DatasetID, since time.Time) ([]food.ID, error) {
	var out []food.ID
	err := r.q.each(ctx,
		`SELECT i.food_id FROM source_items i JOIN foods f ON f.id = i.food_id
		WHERE i.dataset_id = ? AND i.fetched_at >= ? AND f.discontinued_at IS NULL`,
		[]any{id, r.d.timeValue(since)},
		func(rows *sql.Rows) error {
			var fid food.ID
			out = append(out, fid)
			return rows.Scan(&out[len(out)-1])
		})
	return out, err
}

func (r sources) MarkMissing(ctx context.Context, id food.DatasetID, seenBefore time.Time, limit int) ([]food.ID, error) {
	before := r.d.timeValue(seenBefore)
	if _, err := r.q.exec(ctx,
		"UPDATE source_items SET misses = misses + 1 WHERE dataset_id = ? AND seen_at < ?", id, before); err != nil {
		return nil, err
	}
	var out []food.ID
	err := r.q.each(ctx,
		"SELECT food_id FROM source_items WHERE dataset_id = ? AND seen_at < ? AND misses >= ? ORDER BY food_id",
		[]any{id, before, limit},
		func(rows *sql.Rows) error {
			var fid food.ID
			out = append(out, fid)
			return rows.Scan(&out[len(out)-1])
		})
	return out, err
}

func (r sources) CacheGet(ctx context.Context, id food.DatasetID, key string, now time.Time) ([]byte, error) {
	var body []byte
	err := r.q.queryRow(ctx,
		"SELECT body FROM source_cache WHERE dataset_id = ? AND key = ? AND expires_at > ?",
		id, key, r.d.timeValue(now)).Scan(&body)
	return body, notFound(err)
}

func (r sources) CachePut(ctx context.Context, id food.DatasetID, key string, body []byte, expires time.Time) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO source_cache (dataset_id, key, body, expires_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (dataset_id, key) DO UPDATE SET body = excluded.body, expires_at = excluded.expires_at`,
		id, key, body, r.d.timeValue(expires))
	return err
}

func (r sources) CachePurge(ctx context.Context, now time.Time) error {
	_, err := r.q.exec(ctx, "DELETE FROM source_cache WHERE expires_at <= ?", r.d.timeValue(now))
	return err
}
