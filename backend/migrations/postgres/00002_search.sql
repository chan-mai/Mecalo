-- +goose NO TRANSACTION
-- +goose Up
ALTER TABLE food_revisions ADD COLUMN aliases TEXT NOT NULL DEFAULT '';
ALTER TABLE food_revisions ADD COLUMN reading_key TEXT NOT NULL DEFAULT '';
CREATE INVERTED INDEX food_revisions_reading_idx ON food_revisions (reading_key gin_trgm_ops);

CREATE TABLE food_query_links (
	user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	query_key    TEXT NOT NULL,
	food_id      TEXT NOT NULL REFERENCES foods (id) ON DELETE CASCADE,
	uses         INT NOT NULL,
	last_used_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, query_key, food_id)
);

-- 検索キーの生成規則の版
CREATE TABLE store_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

-- +goose Down
DROP TABLE store_meta;
DROP TABLE food_query_links;
DROP INDEX food_revisions@food_revisions_reading_idx;
ALTER TABLE food_revisions DROP COLUMN reading_key;
ALTER TABLE food_revisions DROP COLUMN aliases;
