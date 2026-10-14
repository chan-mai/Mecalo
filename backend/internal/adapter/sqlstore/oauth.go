package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type oauth repos

func (r oauth) CreateClient(ctx context.Context, c port.OAuthClient) error {
	uris, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return err
	}
	_, err = r.q.exec(ctx,
		"INSERT INTO oauth_clients (id, name, redirect_uris, secret_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		c.ID, c.Name, string(uris), c.SecretHash, r.d.timeValue(c.CreatedAt))
	return err
}

func (r oauth) GetClient(ctx context.Context, id string) (port.OAuthClient, error) {
	c := port.OAuthClient{ID: id}
	var uris string
	err := r.q.queryRow(ctx, "SELECT name, redirect_uris, secret_hash, created_at FROM oauth_clients WHERE id = ?", id).
		Scan(&c.Name, &uris, &c.SecretHash, timeCol{&c.CreatedAt})
	if err != nil {
		return port.OAuthClient{}, notFound(err)
	}
	return c, json.Unmarshal([]byte(uris), &c.RedirectURIs)
}

func (r oauth) DeleteUnusedClients(ctx context.Context, createdBefore time.Time) error {
	_, err := r.q.exec(ctx, `
		DELETE FROM oauth_clients WHERE created_at < ?
		AND NOT EXISTS (SELECT 1 FROM oauth_grants g WHERE g.client_id = oauth_clients.id)
		AND NOT EXISTS (SELECT 1 FROM oauth_codes c WHERE c.client_id = oauth_clients.id)`,
		r.d.timeValue(createdBefore))
	return err
}

func (r oauth) CreateCode(ctx context.Context, c port.OAuthCode) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO oauth_codes (hash, client_id, client_name, user_id, redirect_uri, challenge, scope, resource, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Hash, c.ClientID, c.ClientName, c.UserID, c.RedirectURI, c.Challenge, c.Scope, c.Resource, r.d.timeValue(c.ExpiresAt))
	return err
}

func (r oauth) TakeCode(ctx context.Context, hash string) (port.OAuthCode, error) {
	c := port.OAuthCode{Hash: hash}
	err := r.q.queryRow(ctx, `
		SELECT client_id, client_name, user_id, redirect_uri, challenge, scope, resource, expires_at
		FROM oauth_codes WHERE hash = ?`, hash).
		Scan(&c.ClientID, &c.ClientName, &c.UserID, &c.RedirectURI, &c.Challenge, &c.Scope, &c.Resource, timeCol{&c.ExpiresAt})
	if err != nil {
		return port.OAuthCode{}, notFound(err)
	}
	// 同時の交換は片方のみ成功
	if err := affectedOne(r.q.exec(ctx, "DELETE FROM oauth_codes WHERE hash = ?", hash)); err != nil {
		return port.OAuthCode{}, err
	}
	return c, nil
}

func (r oauth) CreateGrant(ctx context.Context, g port.OAuthGrant) error {
	_, err := r.q.exec(ctx, `
		INSERT INTO oauth_grants (id, user_id, client_id, client_name, scope, resource, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.UserID, g.ClientID, g.ClientName, g.Scope, g.Resource, r.d.timeValue(g.CreatedAt))
	return err
}

func (r oauth) ListGrants(ctx context.Context, owner user.ID) ([]port.OAuthGrant, error) {
	var out []port.OAuthGrant
	err := r.q.each(ctx, `
		SELECT id, client_id, client_name, scope, resource, created_at FROM oauth_grants
		WHERE user_id = ? ORDER BY created_at, id`, []any{owner}, func(rows *sql.Rows) error {
		g := port.OAuthGrant{UserID: owner}
		if err := rows.Scan(&g.ID, &g.ClientID, &g.ClientName, &g.Scope, &g.Resource, timeCol{&g.CreatedAt}); err != nil {
			return err
		}
		out = append(out, g)
		return nil
	})
	return out, err
}

func (r oauth) DeleteGrant(ctx context.Context, owner user.ID, id string) error {
	_, err := r.q.exec(ctx, `
		DELETE FROM oauth_codes WHERE EXISTS (
			SELECT 1 FROM oauth_grants g WHERE g.id = ? AND g.user_id = ?
			AND g.user_id = oauth_codes.user_id AND g.client_id = oauth_codes.client_id AND g.resource = oauth_codes.resource)`, id, owner)
	if err != nil {
		return err
	}
	if _, err := r.q.exec(ctx, "DELETE FROM oauth_tokens WHERE grant_id IN (SELECT id FROM oauth_grants WHERE id = ? AND user_id = ?)", id, owner); err != nil {
		return err
	}
	return affectedOne(r.q.exec(ctx, "DELETE FROM oauth_grants WHERE id = ? AND user_id = ?", id, owner))
}

func (r oauth) DeleteGrantsOf(ctx context.Context, owner user.ID, clientID, resource string) error {
	const match = "user_id = ? AND client_id = ? AND resource = ?"
	if _, err := r.q.exec(ctx, "DELETE FROM oauth_tokens WHERE grant_id IN (SELECT id FROM oauth_grants WHERE "+match+")", owner, clientID, resource); err != nil {
		return err
	}
	_, err := r.q.exec(ctx, "DELETE FROM oauth_grants WHERE "+match, owner, clientID, resource)
	return err
}

func (r oauth) CreateToken(ctx context.Context, t port.OAuthToken) error {
	_, err := r.q.exec(ctx, "INSERT INTO oauth_tokens (hash, grant_id, kind, scope, expires_at) VALUES (?, ?, ?, ?, ?)",
		t.Hash, t.GrantID, t.Kind, t.Scope, r.d.timeValue(t.ExpiresAt))
	return err
}

func (r oauth) FindToken(ctx context.Context, hash string) (port.OAuthToken, port.OAuthGrant, error) {
	t := port.OAuthToken{Hash: hash}
	var g port.OAuthGrant
	err := r.q.queryRow(ctx, `
		SELECT t.grant_id, t.kind, t.scope, t.expires_at, g.user_id, g.client_id, g.client_name, g.scope, g.resource, g.created_at
		FROM oauth_tokens t JOIN oauth_grants g ON g.id = t.grant_id WHERE t.hash = ?`, hash).
		Scan(&t.GrantID, &t.Kind, &t.Scope, timeCol{&t.ExpiresAt}, &g.UserID, &g.ClientID, &g.ClientName, &g.Scope, &g.Resource, timeCol{&g.CreatedAt})
	if err != nil {
		return port.OAuthToken{}, port.OAuthGrant{}, notFound(err)
	}
	g.ID = t.GrantID
	return t, g, nil
}

func (r oauth) UseRefreshToken(ctx context.Context, hash string) error {
	return affectedOne(r.q.exec(ctx, "UPDATE oauth_tokens SET kind = ? WHERE hash = ? AND kind = ?",
		port.OAuthRefreshUsed, hash, port.OAuthRefresh))
}

func (r oauth) DeleteExpired(ctx context.Context, now time.Time) error {
	at := r.d.timeValue(now)
	if _, err := r.q.exec(ctx, "DELETE FROM oauth_codes WHERE expires_at <= ?", at); err != nil {
		return err
	}
	_, err := r.q.exec(ctx, "DELETE FROM oauth_tokens WHERE expires_at <= ?", at)
	return err
}
