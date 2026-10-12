package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	ProviderLocal      = "local"
	SessionTTL         = 30 * 24 * time.Hour
	minPasswordLength  = 8
	maxDisplayNameSize = 64
)

var loginNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

type Accounts struct {
	d   Deps
	opt Options
}

type SignUpInput struct {
	LoginName   string
	DisplayName string
	Password    string
}

// 発行済みセッション, Tokenは利用者へ渡す値
type IssuedSession struct {
	Token     string
	User      user.User
	ExpiresAt time.Time
}

func (s *Accounts) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.d.Store.Users().Count(ctx)
	return n == 0, err
}

func (s *Accounts) RegistrationOpen() bool {
	return s.opt.RegistrationOpen
}

// 利用者0件時のみ, トークン一致時に管理者を作成
func (s *Accounts) Setup(ctx context.Context, token string, in SignUpInput) (IssuedSession, error) {
	if s.d.SetupToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.d.SetupToken)) != 1 {
		return IssuedSession{}, ErrForbidden
	}
	var u user.User
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		n, err := r.Users().Count(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrConflict
		}
		u, err = s.createLocal(ctx, r, in, user.RoleAdmin)
		return err
	})
	if err != nil {
		return IssuedSession{}, err
	}
	return s.issue(ctx, u)
}

func (s *Accounts) SignUp(ctx context.Context, in SignUpInput) (IssuedSession, error) {
	if !s.opt.RegistrationOpen {
		return IssuedSession{}, ErrForbidden
	}
	var u user.User
	err := s.d.Store.InTx(ctx, func(r port.Repositories) (err error) {
		u, err = s.createLocal(ctx, r, in, user.RoleMember)
		return err
	})
	if err != nil {
		return IssuedSession{}, err
	}
	return s.issue(ctx, u)
}

// 管理者による利用者作成
func (s *Accounts) CreateUser(ctx context.Context, actor user.User, in SignUpInput, role user.Role) (user.User, error) {
	if !actor.IsAdmin() {
		return user.User{}, ErrForbidden
	}
	return s.CreateUserUnchecked(ctx, in, role)
}

// CLI用, 権限確認なし
func (s *Accounts) CreateUserUnchecked(ctx context.Context, in SignUpInput, role user.Role) (user.User, error) {
	var u user.User
	err := s.d.Store.InTx(ctx, func(r port.Repositories) (err error) {
		u, err = s.createLocal(ctx, r, in, role)
		return err
	})
	return u, err
}

func (s *Accounts) ListUsers(ctx context.Context, actor user.User) ([]user.User, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	return s.d.Store.Users().List(ctx)
}

func (s *Accounts) DeleteUser(ctx context.Context, actor user.User, id user.ID) error {
	if !actor.IsAdmin() || actor.ID == id {
		return ErrForbidden
	}
	return s.d.Store.Users().Delete(ctx, id)
}

func (s *Accounts) LoginLocal(ctx context.Context, loginName, password string) (IssuedSession, error) {
	u, cred, err := s.d.Store.Users().FindByCredential(ctx, ProviderLocal, strings.ToLower(loginName))
	if errors.Is(err, ErrNotFound) {
		return IssuedSession{}, ErrUnauthorized
	}
	if err != nil {
		return IssuedSession{}, err
	}
	ok, err := s.d.Hasher.Verify(cred.PasswordHash, password)
	if err != nil {
		return IssuedSession{}, err
	}
	if !ok {
		return IssuedSession{}, ErrUnauthorized
	}
	return s.issue(ctx, u)
}

func (s *Accounts) ChangePassword(ctx context.Context, u user.User, current, next string) error {
	if len(next) < minPasswordLength {
		return invalid(errors.New("password too short"))
	}
	return s.d.Store.InTx(ctx, func(r port.Repositories) error {
		cred, err := r.Users().LocalCredential(ctx, u.ID)
		if errors.Is(err, ErrNotFound) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		ok, err := s.d.Hasher.Verify(cred.PasswordHash, current)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUnauthorized
		}
		hash, err := s.d.Hasher.Hash(next)
		if err != nil {
			return err
		}
		return r.Users().UpdatePassword(ctx, u.ID, hash)
	})
}

type ExternalIdentity struct {
	Issuer      string
	Subject     string
	DisplayName string
}

// 外部IDに紐づく利用者でログイン, 未登録時は設定に応じ作成
func (s *Accounts) LoginExternal(ctx context.Context, id ExternalIdentity) (IssuedSession, error) {
	var u user.User
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		found, _, err := r.Users().FindByCredential(ctx, id.Issuer, id.Subject)
		if err == nil {
			u = found
			return nil
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		n, err := r.Users().Count(ctx)
		if err != nil {
			return err
		}
		role := user.RoleMember
		switch {
		case n == 0:
			role = user.RoleAdmin
		case !s.opt.OIDCAutoProvision:
			return ErrForbidden
		}
		u = user.User{
			ID:          user.ID(s.d.IDs.NewID()),
			DisplayName: trimDisplayName(id.DisplayName, id.Subject),
			Role:        role,
			CreatedAt:   s.d.Clock.Now(),
		}
		return r.Users().Create(ctx, u, port.Credential{UserID: u.ID, Provider: id.Issuer, Subject: id.Subject})
	})
	if err != nil {
		return IssuedSession{}, err
	}
	return s.issue(ctx, u)
}

func (s *Accounts) Authenticate(ctx context.Context, token string) (user.User, error) {
	if token == "" {
		return user.User{}, ErrUnauthorized
	}
	sess, err := s.d.Store.Sessions().Get(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return user.User{}, ErrUnauthorized
	}
	if err != nil {
		return user.User{}, err
	}
	if !s.d.Clock.Now().Before(sess.ExpiresAt) {
		return user.User{}, ErrUnauthorized
	}
	u, err := s.d.Store.Users().Get(ctx, sess.UserID)
	if errors.Is(err, ErrNotFound) {
		return user.User{}, ErrUnauthorized
	}
	return u, err
}

func (s *Accounts) Logout(ctx context.Context, token string) error {
	return s.d.Store.Sessions().Delete(ctx, hashToken(token))
}

func (s *Accounts) PurgeExpiredSessions(ctx context.Context) error {
	return s.d.Store.Sessions().DeleteExpired(ctx, s.d.Clock.Now())
}

func (s *Accounts) createLocal(ctx context.Context, r port.Repositories, in SignUpInput, role user.Role) (user.User, error) {
	login := strings.ToLower(strings.TrimSpace(in.LoginName))
	if !loginNamePattern.MatchString(login) {
		return user.User{}, invalid(errors.New("login name must be 3-32 characters of a-z, 0-9, _ . -"))
	}
	if len(in.Password) < minPasswordLength {
		return user.User{}, invalid(errors.New("password too short"))
	}
	if _, _, err := r.Users().FindByCredential(ctx, ProviderLocal, login); err == nil {
		return user.User{}, ErrConflict
	} else if !errors.Is(err, ErrNotFound) {
		return user.User{}, err
	}
	hash, err := s.d.Hasher.Hash(in.Password)
	if err != nil {
		return user.User{}, err
	}
	u := user.User{
		ID:          user.ID(s.d.IDs.NewID()),
		DisplayName: trimDisplayName(in.DisplayName, login),
		Role:        role,
		CreatedAt:   s.d.Clock.Now(),
	}
	cred := port.Credential{UserID: u.ID, Provider: ProviderLocal, Subject: login, PasswordHash: hash}
	if err := r.Users().Create(ctx, u, cred); err != nil {
		return user.User{}, err
	}
	return u, nil
}

func (s *Accounts) issue(ctx context.Context, u user.User) (IssuedSession, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return IssuedSession{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	now := s.d.Clock.Now()
	sess := port.Session{ID: hashToken(token), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(SessionTTL)}
	if err := s.d.Store.Sessions().Create(ctx, sess); err != nil {
		return IssuedSession{}, err
	}
	return IssuedSession{Token: token, User: u, ExpiresAt: sess.ExpiresAt}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func trimDisplayName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	if r := []rune(name); len(r) > maxDisplayNameSize {
		name = string(r[:maxDisplayNameSize])
	}
	return name
}
