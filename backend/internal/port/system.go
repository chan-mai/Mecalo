package port

import (
	"context"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/food"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

// バーコードからの外部食品照会
type BarcodeLookup interface {
	Lookup(ctx context.Context, barcode string) (ExternalProduct, error)
	Dataset() food.Dataset
}

type ExternalProduct struct {
	Barcode  string
	Revision food.Revision
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}

// 日本語の読み(カタカナ)への変換
type TextReader interface {
	Reading(text string) string
}
