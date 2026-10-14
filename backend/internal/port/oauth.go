package port

import (
	"context"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

// 動的登録したOAuthクライアント
type OAuthClient struct {
	ID           string
	Name         string
	RedirectURIs []string
	// 空は公開クライアント
	SecretHash string
	CreatedAt  time.Time
}

// 未使用の認可コード
type OAuthCode struct {
	Hash        string
	ClientID    string
	ClientName  string
	UserID      user.ID
	RedirectURI string
	// PKCEのS256チャレンジ
	Challenge string
	Scope     user.Scope
	Resource  string
	ExpiresAt time.Time
}

// 利用者が許可したクライアント
type OAuthGrant struct {
	ID         string
	UserID     user.ID
	ClientID   string
	ClientName string
	Scope      user.Scope
	Resource   string
	CreatedAt  time.Time
}

type OAuthTokenKind string

const (
	OAuthAccess  OAuthTokenKind = "access"
	OAuthRefresh OAuthTokenKind = "refresh"
	// 再利用検知用に期限まで保持
	OAuthRefreshUsed OAuthTokenKind = "refresh_used"
)

type OAuthToken struct {
	Hash    string
	GrantID string
	Kind    OAuthTokenKind
	// 許可より狭い範囲, 空で許可状態と等価j
	Scope     user.Scope
	ExpiresAt time.Time
}

type OAuthRepository interface {
	CreateClient(ctx context.Context, c OAuthClient) error
	GetClient(ctx context.Context, id string) (OAuthClient, error)
	// 許可のないまま期限を過ぎたクライアント
	DeleteUnusedClients(ctx context.Context, createdBefore time.Time) error
	CreateCode(ctx context.Context, c OAuthCode) error
	// 取得と同時に削除, 2回目以降はErrNotFound
	TakeCode(ctx context.Context, hash string) (OAuthCode, error)
	CreateGrant(ctx context.Context, g OAuthGrant) error
	// 作成順
	ListGrants(ctx context.Context, owner user.ID) ([]OAuthGrant, error)
	// トークン,未使用コード 削除
	DeleteGrant(ctx context.Context, owner user.ID, id string) error
	DeleteGrantsOf(ctx context.Context, owner user.ID, clientID, resource string) error
	CreateToken(ctx context.Context, t OAuthToken) error
	FindToken(ctx context.Context, hash string) (OAuthToken, OAuthGrant, error)
	// 未使用のリフレッシュトークンを使用済み, 該当なしErrNotFound
	UseRefreshToken(ctx context.Context, hash string) error
	// 期限切れのコードとトークン
	DeleteExpired(ctx context.Context, now time.Time) error
}

// Client ID Metadata Documentの内容
type ClientMetadata struct {
	ClientID     string
	Name         string
	RedirectURIs []string
}

// URL形式のclient_idのメタデータ取得
type ClientMetadataFetcher interface {
	Fetch(ctx context.Context, clientID string) (ClientMetadata, error)
}
