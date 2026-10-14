package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	OAuthAccessTTL  = time.Hour
	OAuthRefreshTTL = 30 * 24 * time.Hour
	oauthCodeTTL    = 5 * time.Minute
	// 許可のないまま残す動的登録クライアントの期間
	oauthUnusedClientTTL = 24 * time.Hour
	maxClientName        = 100
	maxRedirectURIs      = 10
	accessTokenPrefix    = "mecalo_at_"
	refreshTokenPrefix   = "mecalo_rt_"
	// MCPエンドポイントのパス
	MCPPath = "/mcp"
)

// RFC 6749のエラー応答
type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	return e.Code + ": " + e.Description
}

func oauthErr(code, desc string) error {
	return &OAuthError{Code: code, Description: desc}
}

// MCPの認可サーバ
type OAuth struct {
	d    Deps
	live *live
}

// 公開URLとMCPが有効な場合のみ
func (s *OAuth) Enabled() bool {
	l := s.live.get()
	return l.MCPEnabled && l.PublicURL != ""
}

func (s *OAuth) Issuer() string {
	return s.live.get().PublicURL
}

// トークンの対象リソース
func (s *OAuth) Resource() string {
	return s.live.get().PublicURL + MCPPath
}

type OAuthRegistration struct {
	Name         string
	RedirectURIs []string
	// none, client_secret_basic, client_secret_post
	AuthMethod string
}

type RegisteredClient struct {
	port.OAuthClient
	AuthMethod string
	// 公開クライアントは空
	Secret string
}

// RFC 7591
func (s *OAuth) Register(ctx context.Context, in OAuthRegistration) (RegisteredClient, error) {
	if !s.Enabled() {
		return RegisteredClient{}, ErrDisabled
	}
	if len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > maxRedirectURIs {
		return RegisteredClient{}, oauthErr("invalid_redirect_uri", "1-10 redirect_uris are required")
	}
	for _, u := range in.RedirectURIs {
		if err := validRedirectURI(u); err != nil {
			return RegisteredClient{}, oauthErr("invalid_redirect_uri", err.Error())
		}
	}
	method := in.AuthMethod
	if method == "" {
		method = "client_secret_basic"
	}
	if !slices.Contains([]string{"none", "client_secret_basic", "client_secret_post"}, method) {
		return RegisteredClient{}, oauthErr("invalid_client_metadata", "unsupported token_endpoint_auth_method")
	}
	c := port.OAuthClient{
		ID:           s.d.IDs.NewID(),
		Name:         truncate(strings.TrimSpace(in.Name), maxClientName),
		RedirectURIs: in.RedirectURIs,
		CreatedAt:    s.d.Clock.Now(),
	}
	out := RegisteredClient{AuthMethod: method}
	if method != "none" {
		secret, err := randomSecret("")
		if err != nil {
			return RegisteredClient{}, err
		}
		c.SecretHash = hashToken(secret)
		out.Secret = secret
	}
	if err := s.d.Store.OAuth().CreateClient(ctx, c); err != nil {
		return RegisteredClient{}, err
	}
	out.OAuthClient = c
	return out, nil
}

// 認可リクエストのパラメータ
type AuthorizationRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	State               string
	Resource            string
}

// 同意画面に表示する内容
type Consent struct {
	ClientID   string
	ClientName string
	// Client ID Metadata Documentのホスト, 動的登録は空
	ClientHost   string
	RedirectURI  string
	RedirectHost string
	// localhostへのリダイレクト
	Loopback bool
	// 要求された範囲
	Scope user.Scope
}

// 利用者へ表示せずクライアントへ返すエラー
type AuthorizationRedirect struct {
	URL string
}

func (e *AuthorizationRedirect) Error() string {
	return "authorization error redirect"
}

type resolvedClient struct {
	id           string
	name         string
	host         string
	redirectURIs []string
	secretHash   string
}

// クライアントとリダイレクト先が不正ならErrInvalid, それ以外の不正は*AuthorizationRedirect
func (s *OAuth) Check(ctx context.Context, req AuthorizationRequest) (Consent, error) {
	if !s.Enabled() {
		return Consent{}, ErrDisabled
	}
	c, err := s.client(ctx, req.ClientID)
	if err != nil {
		return Consent{}, err
	}
	if !redirectAllowed(c.redirectURIs, req.RedirectURI) {
		return Consent{}, Invalid(errors.New("redirect_uri is not registered for this client"))
	}
	fail := func(code, desc string) (Consent, error) {
		return Consent{}, &AuthorizationRedirect{URL: s.errorRedirect(req, code, desc)}
	}
	if req.ResponseType != "code" {
		return fail("unsupported_response_type", "response_type must be code")
	}
	if req.CodeChallengeMethod != "S256" || req.CodeChallenge == "" {
		return fail("invalid_request", "PKCE with S256 is required")
	}
	scope, ok := parseScope(req.Scope)
	if !ok {
		return fail("invalid_scope", "supported scopes are read and write")
	}
	if !s.resourceMatches(req.Resource) {
		return fail("invalid_target", "resource must be "+s.Resource())
	}
	ru, _ := url.Parse(req.RedirectURI)
	return Consent{
		ClientID:     c.id,
		ClientName:   c.name,
		ClientHost:   c.host,
		RedirectURI:  req.RedirectURI,
		RedirectHost: ru.Host,
		Loopback:     isLoopback(ru),
		Scope:        scope,
	}, nil
}

// 同意・拒否の結果のリダイレクト先
func (s *OAuth) Decide(ctx context.Context, u user.User, req AuthorizationRequest, approve bool, scope user.Scope) (string, error) {
	consent, err := s.Check(ctx, req)
	if err != nil {
		return "", err
	}
	if !approve {
		return s.errorRedirect(req, "access_denied", "the user denied the request"), nil
	}
	if scope == "" {
		scope = consent.Scope
	}
	if !scope.Valid() || !consent.Scope.Allows(scope) {
		return "", Invalid(errors.New("scope exceeds the request"))
	}
	code, err := randomSecret("")
	if err != nil {
		return "", err
	}
	err = s.d.Store.OAuth().CreateCode(ctx, port.OAuthCode{
		Hash:        hashToken(code),
		ClientID:    consent.ClientID,
		ClientName:  consent.ClientName,
		UserID:      u.ID,
		RedirectURI: req.RedirectURI,
		Challenge:   req.CodeChallenge,
		Scope:       scope,
		Resource:    s.Resource(),
		ExpiresAt:   s.d.Clock.Now().Add(oauthCodeTTL),
	})
	if err != nil {
		return "", err
	}
	return s.redirect(req, url.Values{"code": {code}}), nil
}

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Resource     string
	Scope        string
}

type IssuedOAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	Scope        user.Scope
}

// トークンエンドポイント, 失敗は*OAuthError
func (s *OAuth) Exchange(ctx context.Context, req TokenRequest) (IssuedOAuthToken, error) {
	if !s.Enabled() {
		return IssuedOAuthToken{}, ErrDisabled
	}
	c, err := s.client(ctx, req.ClientID)
	if errors.Is(err, ErrInvalid) {
		return IssuedOAuthToken{}, oauthErr("invalid_client", "unknown client")
	}
	if err != nil {
		return IssuedOAuthToken{}, err
	}
	if c.secretHash != "" && subtle.ConstantTimeCompare([]byte(hashToken(req.ClientSecret)), []byte(c.secretHash)) != 1 {
		return IssuedOAuthToken{}, oauthErr("invalid_client", "client authentication failed")
	}
	if req.Resource != "" && !s.resourceMatches(req.Resource) {
		return IssuedOAuthToken{}, oauthErr("invalid_target", "resource must be "+s.Resource())
	}
	switch req.GrantType {
	case "authorization_code":
		return s.exchangeCode(ctx, c, req)
	case "refresh_token":
		return s.refresh(ctx, c, req)
	}
	return IssuedOAuthToken{}, oauthErr("unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
}

func (s *OAuth) exchangeCode(ctx context.Context, c resolvedClient, req TokenRequest) (IssuedOAuthToken, error) {
	var out IssuedOAuthToken
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		code, err := r.OAuth().TakeCode(ctx, hashToken(req.Code))
		if errors.Is(err, ErrNotFound) {
			return oauthErr("invalid_grant", "unknown or used code")
		}
		if err != nil {
			return err
		}
		switch {
		case !s.d.Clock.Now().Before(code.ExpiresAt):
			return oauthErr("invalid_grant", "code expired")
		case code.ClientID != c.id:
			return oauthErr("invalid_grant", "code was issued to another client")
		case code.RedirectURI != req.RedirectURI:
			return oauthErr("invalid_grant", "redirect_uri mismatch")
		case !pkceMatches(req.CodeVerifier, code.Challenge):
			return oauthErr("invalid_grant", "code_verifier mismatch")
		case code.Resource != s.Resource():
			return oauthErr("invalid_grant", "code was issued for another resource")
		}
		if err := r.OAuth().DeleteGrantsOf(ctx, code.UserID, code.ClientID, code.Resource); err != nil {
			return err
		}
		g := port.OAuthGrant{
			ID:         s.d.IDs.NewID(),
			UserID:     code.UserID,
			ClientID:   code.ClientID,
			ClientName: code.ClientName,
			Scope:      code.Scope,
			Resource:   code.Resource,
			CreatedAt:  s.d.Clock.Now(),
		}
		if err := r.OAuth().CreateGrant(ctx, g); err != nil {
			return err
		}
		out, err = s.issue(ctx, r, g, g.Scope)
		return err
	})
	return out, err
}

// リフレッシュトークンは1回限り, 再利用時は許可ごと失効
func (s *OAuth) refresh(ctx context.Context, c resolvedClient, req TokenRequest) (IssuedOAuthToken, error) {
	var out IssuedOAuthToken
	reused := false
	err := s.d.Store.InTx(ctx, func(r port.Repositories) error {
		hash := hashToken(req.RefreshToken)
		t, g, err := r.OAuth().FindToken(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			return oauthErr("invalid_grant", "unknown refresh token")
		}
		if err != nil {
			return err
		}
		switch {
		case g.ClientID != c.id:
			return oauthErr("invalid_grant", "refresh token was issued to another client")
		case t.Kind == port.OAuthRefreshUsed:
			reused = true
			return r.OAuth().DeleteGrant(ctx, g.UserID, g.ID)
		case t.Kind != port.OAuthRefresh || !s.d.Clock.Now().Before(t.ExpiresAt):
			return oauthErr("invalid_grant", "refresh token expired")
		case g.Resource != s.Resource():
			return oauthErr("invalid_grant", "refresh token was issued for another resource")
		}
		granted := tokenScope(t, g)
		scope := granted
		if req.Scope != "" {
			var ok bool
			scope, ok = parseScope(req.Scope)
			if !ok || !granted.Allows(scope) {
				return oauthErr("invalid_scope", "scope exceeds the grant")
			}
		}
		err = r.OAuth().UseRefreshToken(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			reused = true
			return r.OAuth().DeleteGrant(ctx, g.UserID, g.ID)
		}
		if err != nil {
			return err
		}
		out, err = s.issue(ctx, r, g, scope)
		return err
	})
	if err == nil && reused {
		return IssuedOAuthToken{}, oauthErr("invalid_grant", "refresh token reuse detected; the grant was revoked")
	}
	return out, err
}

// トークンの範囲, 許可を超えない
func tokenScope(t port.OAuthToken, g port.OAuthGrant) user.Scope {
	if t.Scope == "" || !g.Scope.Allows(t.Scope) {
		return g.Scope
	}
	return t.Scope
}

// アクセストークンはscopeの範囲, リフレッシュトークンは許可の範囲
func (s *OAuth) issue(ctx context.Context, r port.Repositories, g port.OAuthGrant, scope user.Scope) (IssuedOAuthToken, error) {
	access, err := randomSecret(accessTokenPrefix)
	if err != nil {
		return IssuedOAuthToken{}, err
	}
	refresh, err := randomSecret(refreshTokenPrefix)
	if err != nil {
		return IssuedOAuthToken{}, err
	}
	now := s.d.Clock.Now()
	for _, t := range []port.OAuthToken{
		{Hash: hashToken(access), GrantID: g.ID, Kind: port.OAuthAccess, Scope: scope, ExpiresAt: now.Add(OAuthAccessTTL)},
		{Hash: hashToken(refresh), GrantID: g.ID, Kind: port.OAuthRefresh, ExpiresAt: now.Add(OAuthRefreshTTL)},
	} {
		if err := r.OAuth().CreateToken(ctx, t); err != nil {
			return IssuedOAuthToken{}, err
		}
	}
	return IssuedOAuthToken{AccessToken: access, RefreshToken: refresh, ExpiresIn: OAuthAccessTTL, Scope: scope}, nil
}

// アクセストークンの検証, 該当なしはErrUnauthorized
func (s *OAuth) authenticate(ctx context.Context, secret string) (TokenPrincipal, error) {
	if !s.Enabled() || !strings.HasPrefix(secret, accessTokenPrefix) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	t, g, err := s.d.Store.OAuth().FindToken(ctx, hashToken(secret))
	if errors.Is(err, ErrNotFound) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPrincipal{}, err
	}
	if t.Kind != port.OAuthAccess || !s.d.Clock.Now().Before(t.ExpiresAt) || g.Resource != s.Resource() {
		return TokenPrincipal{}, ErrUnauthorized
	}
	u, err := s.d.Store.Users().Get(ctx, g.UserID)
	if errors.Is(err, ErrNotFound) {
		return TokenPrincipal{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPrincipal{}, err
	}
	return TokenPrincipal{User: u, Scope: tokenScope(t, g)}, nil
}

func (s *OAuth) Grants(ctx context.Context, u user.User) ([]port.OAuthGrant, error) {
	return s.d.Store.OAuth().ListGrants(ctx, u.ID)
}

// 他の利用者の許可はErrNotFound
func (s *OAuth) RevokeGrant(ctx context.Context, u user.User, id string) error {
	return s.d.Store.InTx(ctx, func(r port.Repositories) error {
		return r.OAuth().DeleteGrant(ctx, u.ID, id)
	})
}

func (s *OAuth) Purge(ctx context.Context) error {
	now := s.d.Clock.Now()
	if err := s.d.Store.OAuth().DeleteExpired(ctx, now); err != nil {
		return err
	}
	return s.d.Store.OAuth().DeleteUnusedClients(ctx, now.Add(-oauthUnusedClientTTL))
}

// URL形式はClient ID Metadata Document, それ以外は動的登録
func (s *OAuth) client(ctx context.Context, id string) (resolvedClient, error) {
	if strings.HasPrefix(id, "https://") {
		if s.d.ClientMetadata == nil {
			return resolvedClient{}, Invalid(errors.New("client metadata documents are not supported"))
		}
		m, err := s.d.ClientMetadata.Fetch(ctx, id)
		if err != nil {
			return resolvedClient{}, Invalid(fmt.Errorf("client metadata: %w", err))
		}
		for _, u := range m.RedirectURIs {
			if err := validRedirectURI(u); err != nil {
				return resolvedClient{}, Invalid(err)
			}
		}
		cu, _ := url.Parse(id)
		return resolvedClient{id: id, name: truncate(m.Name, maxClientName), host: cu.Host, redirectURIs: m.RedirectURIs}, nil
	}
	c, err := s.d.Store.OAuth().GetClient(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return resolvedClient{}, Invalid(errors.New("unknown client_id"))
	}
	if err != nil {
		return resolvedClient{}, err
	}
	return resolvedClient{id: c.ID, name: c.Name, redirectURIs: c.RedirectURIs, secretHash: c.SecretHash}, nil
}

// 省略時は自身のリソース
func (s *OAuth) resourceMatches(resource string) bool {
	if resource == "" {
		return true
	}
	return canonicalURL(resource) == canonicalURL(s.Resource())
}

func (s *OAuth) redirect(req AuthorizationRequest, params url.Values) string {
	u, _ := url.Parse(req.RedirectURI)
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	if req.State != "" {
		q.Set("state", req.State)
	}
	q.Set("iss", s.Issuer())
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *OAuth) errorRedirect(req AuthorizationRequest, code, desc string) string {
	return s.redirect(req, url.Values{"error": {code}, "error_description": {desc}})
}

// 空はwrite, writeはreadを含む
func parseScope(scope string) (user.Scope, bool) {
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return user.ScopeWrite, true
	}
	out := user.ScopeRead
	for _, f := range fields {
		switch user.Scope(f) {
		case user.ScopeRead:
		case user.ScopeWrite:
			out = user.ScopeWrite
		default:
			return "", false
		}
	}
	return out, true
}

func pkceMatches(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

// HTTPSまたはlocalhostのhttp
func validRedirectURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Fragment != "" {
		return fmt.Errorf("invalid redirect_uri %q", raw)
	}
	if u.Scheme == "https" || (u.Scheme == "http" && isLoopback(u)) {
		return nil
	}
	return fmt.Errorf("redirect_uri must use https or localhost: %q", raw)
}

func isLoopback(u *url.URL) bool {
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// 完全一致, localhostのhttpはポートを問わない
func redirectAllowed(registered []string, raw string) bool {
	if slices.Contains(registered, raw) {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || !isLoopback(u) {
		return false
	}
	for _, r := range registered {
		ru, err := url.Parse(r)
		if err == nil && ru.Scheme == "http" && ru.Hostname() == u.Hostname() && ru.Path == u.Path && ru.RawQuery == u.RawQuery {
			return true
		}
	}
	return false
}

func canonicalURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

func randomSecret(prefix string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
