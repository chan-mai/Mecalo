package port

import "time"

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}

// 日本語の読み(カタカナ)への変換
type TextReader interface {
	Reading(text string) string
}
