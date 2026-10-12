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
const searchIndexVersion = "4"

type indexer struct {
	reader port.TextReader
}

type searchKeys struct {
	// 表記と読み
	key, reading string
	// 語の区切りの空白を除いた表記と読み, 名前・ブランド・別名の間のみ空白
	compact, compactReading string
	// 項目ごとの表記と読みの2文字の組
	grams []string
}

func (x indexer) keys(name, brand string, aliases []string) searchKeys {
	all := append([]string{name, brand}, aliases...)
	k := searchKeys{key: food.SearchKey(strings.Join(all, " "))}
	k.compact = compactJoin(all, food.SearchKey)
	fields := make([]string, 0, len(all)*2)
	for _, f := range all {
		fields = append(fields, food.SearchKey(f))
	}
	if x.reader != nil {
		readable := append([]string{name}, aliases...)
		k.reading = food.SearchKey(x.reader.Reading(strings.Join(readable, " ")))
		k.compactReading = compactJoin(readable, func(s string) string { return food.SearchKey(x.reader.Reading(s)) })
		for _, f := range readable {
			fields = append(fields, food.SearchKey(x.reader.Reading(f)))
		}
	}
	k.grams = food.BigramsOf(fields...)
	return k
}

// 1文で挿入する組の数
const gramBatch = 400

// リビジョンの組を一括挿入
func insertGrams(ctx context.Context, q conn, revision string, grams []string) error {
	for start := 0; start < len(grams); start += gramBatch {
		batch := grams[start:min(start+gramBatch, len(grams))]
		args := make([]any, 0, len(batch)*2)
		for _, g := range batch {
			args = append(args, g, revision)
		}
		values := strings.TrimSuffix(strings.Repeat("(?, ?), ", len(batch)), ", ")
		if _, err := q.exec(ctx, "INSERT INTO food_search_grams (gram, revision_id) VALUES "+values+" ON CONFLICT DO NOTHING", args...); err != nil {
			return err
		}
	}
	return nil
}

// 項目ごとに空白を除いて空白で連結
func compactJoin(parts []string, key func(string) string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if c := strings.ReplaceAll(key(p), " ", ""); c != "" {
			out = append(out, c)
		}
	}
	return strings.Join(out, " ")
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
		current         bool
	}
	var rows []row
	err = s.q.each(ctx, `
		SELECT r.id, r.name, r.brand, r.aliases, CASE WHEN f.current_revision_id = r.id THEN 1 ELSE 0 END
		FROM food_revisions r JOIN foods f ON f.id = r.food_id`, nil, func(r *sql.Rows) error {
		var (
			x       row
			current int
		)
		if err := r.Scan(&x.id, &x.name, &x.brand, aliasesCol{&x.aliases}, &current); err != nil {
			return err
		}
		x.current = current == 1
		rows = append(rows, x)
		return nil
	})
	if err != nil {
		return err
	}
	if _, err := s.q.exec(ctx, "DELETE FROM food_search_grams"); err != nil {
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
			k := s.idx.keys(x.name, x.brand, x.aliases)
			if _, err := c.exec(ctx,
				"UPDATE food_revisions SET search_key = ?, reading_key = ?, compact_key = ?, compact_reading = ?, gram_count = ? WHERE id = ?",
				k.key, k.reading, k.compact, k.compactReading, len(k.grams), x.id); err != nil {
				_ = tx.Rollback()
				return err
			}
			if x.current {
				if err := insertGrams(ctx, c, x.id, k.grams); err != nil {
					_ = tx.Rollback()
					return err
				}
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
