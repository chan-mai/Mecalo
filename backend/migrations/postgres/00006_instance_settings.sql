-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE instance_settings (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE instance_settings;
