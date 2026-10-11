package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

// 検索キーの生成規則を変えたら上げる
const searchIndexVersion = "2"

type indexer struct {
	reader port.TextReader
}

// 表記の検索キーと読みの検索キー
func (x indexer) keys(name, brand string, aliases []string) (string, string) {
	key := food.SearchKey(strings.Join(append([]string{name, brand}, aliases...), " "))
	if x.reader == nil {
		return key, ""
	}
	reading := food.SearchKey(x.reader.Reading(strings.Join(append([]string{name}, aliases...), " ")))
	return key, reading
}

func (x indexer) version() string {
	if x.reader == nil {
		return searchIndexVersion
	}
	return searchIndexVersion + "+reading"
}

func aliasesArg(aliases []string) string {
	return strings.Join(aliases, "\n")
}

type aliasesCol struct{ a *[]string }

func (c aliasesCol) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	}
	*c.a = nil
	for _, a := range strings.Split(s, "\n") {
		if a != "" {
			*c.a = append(*c.a, a)
		}
	}
	return nil
}

// 生成規則の版が保存値と異なれば全リビジョンの検索キーを再計算
func (s *Store) reindex(ctx context.Context) error {
	var current string
	err := s.q.queryRow(ctx, "SELECT value FROM store_meta WHERE key = 'search_index'").Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if current == s.idx.version() {
		return nil
	}

	type row struct {
		id, name, brand string
		aliases         []string
	}
	var rows []row
	err = s.q.each(ctx, "SELECT id, name, brand, aliases FROM food_revisions", nil, func(r *sql.Rows) error {
		var x row
		if err := r.Scan(&x.id, &x.name, &x.brand, aliasesCol{&x.aliases}); err != nil {
			return err
		}
		rows = append(rows, x)
		return nil
	})
	if err != nil {
		return err
	}
	for start := 0; start < len(rows); start += chunkSize {
		batch := rows[start:min(start+chunkSize, len(rows))]
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		c := conn{db: tx, d: s.d}
		for _, x := range batch {
			key, reading := s.idx.keys(x.name, x.brand, x.aliases)
			if _, err := c.exec(ctx, "UPDATE food_revisions SET search_key = ?, reading_key = ? WHERE id = ?", key, reading, x.id); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	_, err = s.q.exec(ctx, `
		INSERT INTO store_meta (key, value) VALUES ('search_index', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, s.idx.version())
	return err
}

type queryLinks repos

func (r queryLinks) Link(ctx context.Context, owner user.ID, queryKey string, id food.ID, at time.Time) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO food_query_links (user_id, query_key, food_id, uses, last_used_at) VALUES (?, ?, ?, 1, ?)
		ON CONFLICT (user_id, query_key, food_id) DO UPDATE SET
			uses = food_query_links.uses + 1, last_used_at = excluded.last_used_at`,
		owner, queryKey, id, r.d.timeValue(at))
	return err
}

func (r queryLinks) Linked(ctx context.Context, owner user.ID, queryKey string, limit int) ([]food.ID, error) {
	var out []food.ID
	err := r.q.each(ctx, `
		SELECT food_id FROM food_query_links WHERE user_id = ? AND query_key = ?
		ORDER BY uses DESC, last_used_at DESC LIMIT ?`,
		[]any{owner, queryKey, limit},
		func(rows *sql.Rows) error {
			var id food.ID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
			return nil
		})
	return out, err
}
