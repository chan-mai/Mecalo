-- +goose Up
ALTER TABLE oauth_tokens ADD COLUMN scope TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE oauth_tokens DROP COLUMN scope;
