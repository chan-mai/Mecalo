-- +goose NO TRANSACTION
-- +goose Up
-- 語の区切りの空白を除いた検索キー, 値は起動時の再計算で設定
ALTER TABLE food_revisions ADD COLUMN compact_key TEXT NOT NULL DEFAULT '';
ALTER TABLE food_revisions ADD COLUMN compact_reading TEXT NOT NULL DEFAULT '';
CREATE INVERTED INDEX food_revisions_compact_idx ON food_revisions (compact_key gin_trgm_ops);
CREATE INVERTED INDEX food_revisions_compact_reading_idx ON food_revisions (compact_reading gin_trgm_ops);

-- +goose Down
DROP INDEX food_revisions@food_revisions_compact_reading_idx;
DROP INDEX food_revisions@food_revisions_compact_idx;
ALTER TABLE food_revisions DROP COLUMN compact_reading;
ALTER TABLE food_revisions DROP COLUMN compact_key;
