-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE api_tokens (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	name       TEXT NOT NULL,
	scope      TEXT NOT NULL,
	hash       TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX api_tokens_user_idx ON api_tokens (user_id);

-- +goose Down
DROP TABLE api_tokens;
