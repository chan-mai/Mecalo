-- +goose NO TRANSACTION
-- +goose Up
ALTER TABLE foods ADD COLUMN discontinued_at TIMESTAMPTZ;
ALTER TABLE food_revisions ADD COLUMN basis_unit TEXT NOT NULL DEFAULT '';
ALTER TABLE food_revisions ADD COLUMN basis_grams NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE food_revisions ADD COLUMN source_url TEXT NOT NULL DEFAULT '';
ALTER TABLE food_revisions ADD COLUMN retrieved_at TIMESTAMPTZ;

-- 取得元ごとの同期状態
CREATE TABLE source_status (
	dataset_id TEXT PRIMARY KEY,
	cursor     TEXT NOT NULL DEFAULT '',
	synced_at  TIMESTAMPTZ NOT NULL,
	error      TEXT NOT NULL DEFAULT '',
	items      INT NOT NULL DEFAULT 0
);

-- 一覧同期で確認した食品
CREATE TABLE source_items (
	dataset_id TEXT NOT NULL,
	food_id    TEXT NOT NULL REFERENCES foods (id) ON DELETE CASCADE,
	seen_at    TIMESTAMPTZ NOT NULL,
	fetched_at TIMESTAMPTZ NOT NULL,
	misses     INT NOT NULL DEFAULT 0,
	PRIMARY KEY (dataset_id, food_id)
);

-- 外部検索の応答
CREATE TABLE source_cache (
	dataset_id TEXT NOT NULL,
	key        TEXT NOT NULL,
	body       BYTES NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (dataset_id, key)
);
CREATE INDEX source_cache_expires_idx ON source_cache (expires_at);

-- +goose Down
DROP TABLE source_cache;
DROP TABLE source_items;
DROP TABLE source_status;
ALTER TABLE food_revisions DROP COLUMN retrieved_at;
ALTER TABLE food_revisions DROP COLUMN source_url;
ALTER TABLE food_revisions DROP COLUMN basis_grams;
ALTER TABLE food_revisions DROP COLUMN basis_unit;
ALTER TABLE foods DROP COLUMN discontinued_at;
