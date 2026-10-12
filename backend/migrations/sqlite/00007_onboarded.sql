-- +goose Up
ALTER TABLE users ADD COLUMN onboarded INTEGER NOT NULL DEFAULT 0;

-- 設定登録済みの既存利用者は完了扱い
UPDATE users SET onboarded = 1 WHERE id IN (SELECT user_id FROM settings);

-- +goose Down
ALTER TABLE users DROP COLUMN onboarded;
