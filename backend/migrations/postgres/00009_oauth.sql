-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE oauth_clients (
	id            TEXT PRIMARY KEY,
	name          TEXT NOT NULL,
	redirect_uris TEXT NOT NULL,
	secret_hash   TEXT NOT NULL DEFAULT '',
	created_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth_codes (
	hash         TEXT PRIMARY KEY,
	client_id    TEXT NOT NULL,
	client_name  TEXT NOT NULL,
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	redirect_uri TEXT NOT NULL,
	challenge    TEXT NOT NULL,
	scope        TEXT NOT NULL,
	resource     TEXT NOT NULL,
	expires_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth_grants (
	id          TEXT PRIMARY KEY,
	user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	client_id   TEXT NOT NULL,
	client_name TEXT NOT NULL,
	scope       TEXT NOT NULL,
	resource    TEXT NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX oauth_grants_user_idx ON oauth_grants (user_id, client_id);

CREATE TABLE oauth_tokens (
	hash       TEXT PRIMARY KEY,
	grant_id   TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
	kind       TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX oauth_tokens_grant_idx ON oauth_tokens (grant_id);

-- +goose Down
DROP TABLE oauth_tokens;
DROP TABLE oauth_grants;
DROP TABLE oauth_codes;
DROP TABLE oauth_clients;
