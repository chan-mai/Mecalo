package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	// 漏洩時の検出用の接頭辞
	tokenPrefix      = "mecalo_"
	maxTokenName     = 64
	maxTokensPerUser = 20
)

// 外部クライアント用のAPIトークン
type Tokens struct {
	d     Deps
	live  *live
	oauth *OAuth
}

// 発行直後のみ平文を返す
type IssuedToken struct {
	port.Token
	Secret string
}

// トークンで認証した利用者
type TokenPrincipal struct {
	User  user.User
	Scope user.Scope
}

func (s *Tokens) Enabled() bool {
	return s.live.get().MCPEnabled
}

func (s *Tokens) List(ctx context.Context, u user.User) ([]port.Token, error) {
	return s.d.Store.Tokens().List(ctx, u.ID)
}

func (s *Tokens) Issue(ctx context.Context, u user.User, name string, scope user.Scope) (IssuedToken, error) {
	if !s.Enabled() {
		return IssuedToken{}, ErrDisabled
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > maxTokenName {
		return IssuedToken{}, invalid(errors.New("name must be 1-64 characters"))
	}
	if !scope.Valid() {
		return IssuedToken{}, invalid(errors.New("unknown scope"))
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return IssuedToken{}, err
	}
	secret := tokenPrefix + base64.RawURLEncoding.EncodeToString(buf)
	t := port.Token{
		ID:        s.d.IDs.NewID(),
		UserID:    u.ID,
		Name:      name,
		Scope:     scope,
		Hash:      hashToken(secret),
		CreatedAt: s.d.Clock.Now(),
	}
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		list, err := r.Tokens().List(ctx, u.ID)
		if err != nil {
			return err
		}
		if len(list) >= maxTokensPerUser {
			return invalid(errors.New("too many tokens"))
		}
		return r.Tokens().Create(ctx, t)
	})
	if err != nil {
		return IssuedToken{}, err
	}
	return IssuedToken{Token: t, Secret: secret}, nil
}

// 他の利用者のトークンはErrNotFound
func (s *Tokens) Revoke(ctx context.Context, u user.User, id string) error {
	return s.d.Store.Tokens().Delete(ctx, u.ID, id)
}

// APIトークンとOAuthのアクセストークン, 未登録・失効済みはErrUnauthorized
func (s *Tokens) Authenticate(ctx context.Context, secret string) (TokenPrincipal, error) {
	if !s.Enabled() {
		return TokenPrincipal{}, ErrDisabled
	}
	if p, err := s.oauth.authenticate(ctx, secret); !errors.Is(err, ErrUnauthorized) {
		return p, err
	}
	if !strings.HasPrefix(secret, tokenPrefix) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	t, err := s.d.Store.Tokens().FindByHash(ctx, hashToken(secret))
	if errors.Is(err, ErrNotFound) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPrincipal{}, err
	}
	u, err := s.d.Store.Users().Get(ctx, t.UserID)
	if errors.Is(err, ErrNotFound) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPrincipal{}, err
	}
	return TokenPrincipal{User: u, Scope: t.Scope}, nil
}
