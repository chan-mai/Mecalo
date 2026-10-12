-- +goose NO TRANSACTION
-- +goose Up
-- 類似度検索用の2文字の組, 現行リビジョンのみ, 値は起動時の再計算で設定
ALTER TABLE food_revisions ADD COLUMN gram_count INT NOT NULL DEFAULT 0;

CREATE TABLE food_search_grams (
	gram        TEXT NOT NULL,
	revision_id TEXT NOT NULL REFERENCES food_revisions (id) ON DELETE CASCADE,
	PRIMARY KEY (gram, revision_id)
);
CREATE INDEX food_search_grams_revision_idx ON food_search_grams (revision_id);

-- +goose Down
DROP TABLE food_search_grams;
ALTER TABLE food_revisions DROP COLUMN gram_count;
