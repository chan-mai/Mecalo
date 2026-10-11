package sqlstore

import (
	"context"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type users repos

const userColumns = "u.id, u.display_name, u.role, u.created_at"

func scanUser(row interface{ Scan(...any) error }, extra ...any) (user.User, error) {
	var u user.User
	dest := append([]any{&u.ID, &u.DisplayName, &u.Role, timeCol{&u.CreatedAt}}, extra...)
	err := row.Scan(dest...)
	return u, notFound(err)
}

func (r users) Create(ctx context.Context, u user.User, cred port.Credential) error {
	_, err := r.q.exec(ctx,
		"INSERT INTO users (id, display_name, role, created_at) VALUES (?, ?, ?, ?)",
		u.ID, u.DisplayName, u.Role, r.d.timeValue(u.CreatedAt))
	if err != nil {
		return err
	}
	_, err = r.q.exec(ctx,
		"INSERT INTO credentials (provider, subject, user_id, password_hash) VALUES (?, ?, ?, ?)",
		cred.Provider, cred.Subject, u.ID, cred.PasswordHash)
	return err
}

func (r users) Get(ctx context.Context, id user.ID) (user.User, error) {
	return scanUser(r.q.queryRow(ctx, "SELECT "+userColumns+" FROM users u WHERE u.id = ?", id))
}

func (r users) FindByCredential(ctx context.Context, provider, subject string) (user.User, port.Credential, error) {
	cred := port.Credential{Provider: provider, Subject: subject}
	u, err := scanUser(r.q.queryRow(ctx,
		"SELECT "+userColumns+", c.password_hash FROM credentials c JOIN users u ON u.id = c.user_id WHERE c.provider = ? AND c.subject = ?",
		provider, subject), &cred.PasswordHash)
	cred.UserID = u.ID
	return u, cred, err
}

func (r users) LocalCredential(ctx context.Context, id user.ID) (port.Credential, error) {
	cred := port.Credential{UserID: id}
	err := r.q.queryRow(ctx,
		"SELECT provider, subject, password_hash FROM credentials WHERE user_id = ? AND provider = 'local'", id).
		Scan(&cred.Provider, &cred.Subject, &cred.PasswordHash)
	return cred, notFound(err)
}

func (r users) List(ctx context.Context) ([]user.User, error) {
	rows, err := r.q.query(ctx, "SELECT "+userColumns+" FROM users u ORDER BY u.created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r users) Count(ctx context.Context) (int, error) {
	var n int
	err := r.q.queryRow(ctx, "SELECT count(*) FROM users").Scan(&n)
	return n, err
}

func (r users) UpdatePassword(ctx context.Context, id user.ID, hash string) error {
	return affectedOne(r.q.exec(ctx,
		"UPDATE credentials SET password_hash = ? WHERE user_id = ? AND provider = 'local'", hash, id))
}

func (r users) Delete(ctx context.Context, id user.ID) error {
	return affectedOne(r.q.exec(ctx, "DELETE FROM users WHERE id = ?", id))
}

type sessions repos

func (r sessions) Create(ctx context.Context, s port.Session) error {
	_, err := r.q.exec(ctx,
		"INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		s.ID, s.UserID, r.d.timeValue(s.CreatedAt), r.d.timeValue(s.ExpiresAt))
	return err
}

func (r sessions) Get(ctx context.Context, id string) (port.Session, error) {
	s := port.Session{ID: id}
	err := r.q.queryRow(ctx, "SELECT user_id, created_at, expires_at FROM sessions WHERE id = ?", id).
		Scan(&s.UserID, timeCol{&s.CreatedAt}, timeCol{&s.ExpiresAt})
	return s, notFound(err)
}

func (r sessions) Delete(ctx context.Context, id string) error {
	_, err := r.q.exec(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (r sessions) DeleteExpired(ctx context.Context, now time.Time) error {
	_, err := r.q.exec(ctx, "DELETE FROM sessions WHERE expires_at <= ?", r.d.timeValue(now))
	return err
}
