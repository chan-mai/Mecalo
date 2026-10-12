-- +goose NO TRANSACTION
-- +goose Up
ALTER TABLE users ADD COLUMN onboarded BOOL NOT NULL DEFAULT false;

-- 設定登録済みの既存利用者は完了扱い
UPDATE users SET onboarded = true WHERE id IN (SELECT user_id FROM settings);

-- +goose Down
ALTER TABLE users DROP COLUMN onboarded;
