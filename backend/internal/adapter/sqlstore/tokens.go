package sqlstore

import (
	"context"
	"database/sql"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type tokens repos

func (r tokens) Create(ctx context.Context, t port.Token) error {
	_, err := r.q.exec(ctx,
		"INSERT INTO api_tokens (id, user_id, name, scope, hash, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		t.ID, t.UserID, t.Name, t.Scope, t.Hash, r.d.timeValue(t.CreatedAt))
	return err
}

func (r tokens) FindByHash(ctx context.Context, hash string) (port.Token, error) {
	t := port.Token{Hash: hash}
	err := r.q.queryRow(ctx, "SELECT id, user_id, name, scope, created_at FROM api_tokens WHERE hash = ?", hash).
		Scan(&t.ID, &t.UserID, &t.Name, &t.Scope, timeCol{&t.CreatedAt})
	return t, notFound(err)
}

func (r tokens) List(ctx context.Context, owner user.ID) ([]port.Token, error) {
	var out []port.Token
	err := r.q.each(ctx, "SELECT id, name, scope, hash, created_at FROM api_tokens WHERE user_id = ? ORDER BY created_at, id", []any{owner}, func(rows *sql.Rows) error {
		t := port.Token{UserID: owner}
		if err := rows.Scan(&t.ID, &t.Name, &t.Scope, &t.Hash, timeCol{&t.CreatedAt}); err != nil {
			return err
		}
		out = append(out, t)
		return nil
	})
	return out, err
}

func (r tokens) Delete(ctx context.Context, owner user.ID, id string) error {
	return affectedOne(r.q.exec(ctx, "DELETE FROM api_tokens WHERE id = ? AND user_id = ?", id, owner))
}
