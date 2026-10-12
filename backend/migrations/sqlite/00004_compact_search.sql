-- +goose Up
-- 語の区切りの空白を除いた検索キー, 値は起動時の再計算で設定
ALTER TABLE food_revisions ADD COLUMN compact_key TEXT NOT NULL DEFAULT '';
ALTER TABLE food_revisions ADD COLUMN compact_reading TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE food_revisions DROP COLUMN compact_reading;
ALTER TABLE food_revisions DROP COLUMN compact_key;
