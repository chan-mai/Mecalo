package app_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	publicURL = "https://mecalo.example.com"
	verifier  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJ"
	cimdID    = "https://client.example.com/oauth/metadata.json"
)

type fakeMetadata map[string]port.ClientMetadata

func (f fakeMetadata) Fetch(_ context.Context, id string) (port.ClientMetadata, error) {
	m, ok := f[id]
	if !ok {
		return port.ClientMetadata{}, errors.New("status 404")
	}
	return m, nil
}

func challenge(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func newOAuthEnv(t *testing.T, s port.Store) *env {
	t.Helper()
	e, _ := newOAuthEnvClock(t, s)
	return e
}

func newOAuthEnvClock(t *testing.T, s port.Store) (*env, *fixedClock) {
	t.Helper()
	clock := &fixedClock{now: time.Date(2026, 10, 12, 8, 0, 0, 0, jst)}
	e := newEnvWith(t, s, func(d *app.Deps) {
		d.Clock = clock
		d.ClientMetadata = fakeMetadata{cimdID: {
			ClientID: cimdID, Name: "Example Client", RedirectURIs: []string{"http://127.0.0.1/callback"},
		}}
	})
	e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true, PublicURL: publicURL}})
	return e, clock
}

func authRequest(clientID, redirect string) app.AuthorizationRequest {
	return app.AuthorizationRequest{
		ResponseType: "code", ClientID: clientID, RedirectURI: redirect,
		CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256",
		Scope: "read write", State: "xyz", Resource: publicURL + "/mcp",
	}
}

// 許可後のリダイレクト先からコードを取り出す
func codeFrom(t *testing.T, to string) string {
	t.Helper()
	u, err := url.Parse(to)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("state") != "xyz" || q.Get("iss") != publicURL || q.Get("code") == "" {
		t.Fatalf("redirect = %s", to)
	}
	return q.Get("code")
}

func TestOAuthAuthorizationCode(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		redirect := "https://assistant.example.com/callback"
		c, err := e.app.OAuth.Register(e.ctx, app.OAuthRegistration{Name: "Assistant", RedirectURIs: []string{redirect}, AuthMethod: "none"})
		if err != nil {
			t.Fatal(err)
		}
		if c.Secret != "" {
			t.Errorf("public client got a secret")
		}
		req := authRequest(c.ID, redirect)

		consent, err := e.app.OAuth.Check(e.ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if consent.ClientName != "Assistant" || consent.Scope != user.ScopeWrite || consent.RedirectHost != "assistant.example.com" || consent.Loopback {
			t.Errorf("consent = %+v", consent)
		}

		to, err := e.app.OAuth.Decide(e.ctx, e.alice, req, true, user.ScopeRead)
		if err != nil {
			t.Fatal(err)
		}
		code := codeFrom(t, to)

		tokenReq := app.TokenRequest{GrantType: "authorization_code", Code: code, RedirectURI: redirect, CodeVerifier: verifier, ClientID: c.ID, Resource: publicURL + "/mcp"}
		bad := tokenReq
		bad.CodeVerifier = strings.Repeat("x", 43)
		if _, err := e.app.OAuth.Exchange(e.ctx, bad); !isOAuthError(err, "invalid_grant") {
			t.Errorf("wrong verifier: %v", err)
		}
		tok, err := e.app.OAuth.Exchange(e.ctx, tokenReq)
		if err != nil {
			t.Fatal(err)
		}
		if tok.Scope != user.ScopeRead || tok.ExpiresIn != app.OAuthAccessTTL {
			t.Errorf("token = %+v", tok)
		}
		if _, err := e.app.OAuth.Exchange(e.ctx, tokenReq); !isOAuthError(err, "invalid_grant") {
			t.Errorf("code reuse: %v", err)
		}

		p, err := e.app.Tokens.Authenticate(e.ctx, tok.AccessToken)
		if err != nil {
			t.Fatal(err)
		}
		if p.User.ID != e.alice.ID || p.Scope != user.ScopeRead {
			t.Errorf("principal = %+v", p)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.RefreshToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("refresh token used as access token: %v", err)
		}

		refreshed, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: tok.RefreshToken, ClientID: c.ID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: refreshed.RefreshToken, ClientID: c.ID, Scope: "write"}); !isOAuthError(err, "invalid_scope") {
			t.Errorf("scope escalation: %v", err)
		}

		grants, err := e.app.OAuth.Grants(e.ctx, e.alice)
		if err != nil || len(grants) != 1 || grants[0].ClientName != "Assistant" {
			t.Fatalf("grants = %+v, %v", grants, err)
		}
		if err := e.app.OAuth.RevokeGrant(e.ctx, e.bob, grants[0].ID); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("revoke by other user: %v", err)
		}
		if err := e.app.OAuth.RevokeGrant(e.ctx, e.alice, grants[0].ID); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, refreshed.AccessToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("access token after revoke: %v", err)
		}
		if _, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: refreshed.RefreshToken, ClientID: c.ID}); !isOAuthError(err, "invalid_grant") {
			t.Errorf("refresh after revoke: %v", err)
		}
	})
}

func TestOAuthRequestValidation(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		redirect := "https://assistant.example.com/callback"
		c, err := e.app.OAuth.Register(e.ctx, app.OAuthRegistration{Name: "Assistant", RedirectURIs: []string{redirect, "http://localhost:3000/cb"}})
		if err != nil {
			t.Fatal(err)
		}
		if c.Secret == "" || c.AuthMethod != "client_secret_basic" {
			t.Errorf("confidential client = %+v", c)
		}
		for _, uris := range [][]string{nil, {"http://example.com/cb"}, {"myapp://cb"}, {"https://example.com/cb#x"}} {
			if _, err := e.app.OAuth.Register(e.ctx, app.OAuthRegistration{RedirectURIs: uris, AuthMethod: "none"}); !isOAuthError(err, "invalid_redirect_uri") {
				t.Errorf("redirect_uris %v: %v", uris, err)
			}
		}

		// 利用者へ表示するエラー
		for name, req := range map[string]app.AuthorizationRequest{
			"unknown client":      authRequest("nope", redirect),
			"redirect mismatch":   authRequest(c.ID, "https://evil.example.com/callback"),
			"unknown cimd":        authRequest("https://unknown.example.com/meta.json", redirect),
			"cimd wrong redirect": authRequest(cimdID, redirect),
		} {
			if _, err := e.app.OAuth.Check(e.ctx, req); !errors.Is(err, app.ErrInvalid) {
				t.Errorf("%s: %v", name, err)
			}
		}

		// クライアントへ返すエラー
		mutate := map[string]func(*app.AuthorizationRequest){
			"response type": func(r *app.AuthorizationRequest) { r.ResponseType = "token" },
			"no pkce":       func(r *app.AuthorizationRequest) { r.CodeChallenge = "" },
			"plain pkce":    func(r *app.AuthorizationRequest) { r.CodeChallengeMethod = "plain" },
			"scope":         func(r *app.AuthorizationRequest) { r.Scope = "admin" },
			"resource":      func(r *app.AuthorizationRequest) { r.Resource = "https://other.example.com/mcp" },
		}
		for name, fn := range mutate {
			req := authRequest(c.ID, redirect)
			fn(&req)
			_, err := e.app.OAuth.Check(e.ctx, req)
			var r *app.AuthorizationRedirect
			if !errors.As(err, &r) || !strings.HasPrefix(r.URL, redirect+"?") || !strings.Contains(r.URL, "error=") {
				t.Errorf("%s: %v", name, err)
			}
		}

		to, err := e.app.OAuth.Decide(e.ctx, e.alice, authRequest(c.ID, redirect), false, "")
		if err != nil || !strings.Contains(to, "error=access_denied") {
			t.Errorf("deny = %s, %v", to, err)
		}
		readOnly := authRequest(c.ID, redirect)
		readOnly.Scope = "read"
		if _, err := e.app.OAuth.Decide(e.ctx, e.alice, readOnly, true, user.ScopeWrite); !errors.Is(err, app.ErrInvalid) {
			t.Errorf("scope beyond request: %v", err)
		}

		// localhostはポートを問わない
		loop := authRequest(c.ID, "http://localhost:51234/cb")
		consent, err := e.app.OAuth.Check(e.ctx, loop)
		if err != nil || !consent.Loopback {
			t.Fatalf("loopback = %+v, %v", consent, err)
		}
		code := codeFrom(t, mustDecide(t, e, loop))
		base := app.TokenRequest{GrantType: "authorization_code", Code: code, RedirectURI: loop.RedirectURI, CodeVerifier: verifier, ClientID: c.ID}
		if _, err := e.app.OAuth.Exchange(e.ctx, base); !isOAuthError(err, "invalid_client") {
			t.Errorf("missing secret: %v", err)
		}
		base.ClientSecret = c.Secret
		base.RedirectURI = "http://localhost:9/cb"
		if _, err := e.app.OAuth.Exchange(e.ctx, base); !isOAuthError(err, "invalid_grant") {
			t.Errorf("redirect_uri mismatch: %v", err)
		}
	})
}

func TestOAuthClientMetadataDocument(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		req := authRequest(cimdID, "http://127.0.0.1:8123/callback")
		consent, err := e.app.OAuth.Check(e.ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if consent.ClientName != "Example Client" || consent.ClientHost != "client.example.com" || !consent.Loopback {
			t.Errorf("consent = %+v", consent)
		}
		tok, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
			GrantType: "authorization_code", Code: codeFrom(t, mustDecide(t, e, req)), RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
		})
		if err != nil {
			t.Fatal(err)
		}
		if tok.Scope != user.ScopeWrite {
			t.Errorf("scope = %s", tok.Scope)
		}

		// 再認可で前回の許可を置き換え
		if _, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
			GrantType: "authorization_code", Code: codeFrom(t, mustDecide(t, e, req)), RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
		}); err != nil {
			t.Fatal(err)
		}
		if grants, _ := e.app.OAuth.Grants(e.ctx, e.alice); len(grants) != 1 {
			t.Errorf("grants = %+v", grants)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.AccessToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("replaced grant token: %v", err)
		}
	})
}

func TestOAuthResourceAndAvailability(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		req := authRequest(cimdID, "http://127.0.0.1/callback")
		tok, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
			GrantType: "authorization_code", Code: codeFrom(t, mustDecide(t, e, req)), RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
		})
		if err != nil {
			t.Fatal(err)
		}

		// 公開URL変更後の旧リソース向けトークン
		e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true, PublicURL: "https://new.example.com"}})
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.AccessToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("token for old resource: %v", err)
		}
		e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true}})
		if e.app.OAuth.Enabled() {
			t.Error("enabled without public URL")
		}
		if _, err := e.app.OAuth.Check(e.ctx, req); !errors.Is(err, app.ErrDisabled) {
			t.Errorf("check without public URL: %v", err)
		}
		if _, err := e.app.OAuth.Register(e.ctx, app.OAuthRegistration{RedirectURIs: []string{"https://a.example.com/cb"}}); !errors.Is(err, app.ErrDisabled) {
			t.Errorf("register without public URL: %v", err)
		}

		e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true, PublicURL: publicURL}})
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.AccessToken); err != nil {
			t.Errorf("token after restoring URL: %v", err)
		}
	})
}

func TestOAuthExpiry(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e, clock := newOAuthEnvClock(t, s)
		req := authRequest(cimdID, "http://127.0.0.1/callback")
		expired := codeFrom(t, mustDecide(t, e, req))
		code := codeFrom(t, mustDecide(t, e, req))
		exchange := func(code string) (app.IssuedOAuthToken, error) {
			return e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
				GrantType: "authorization_code", Code: code, RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
			})
		}
		tok, err := exchange(code)
		if err != nil {
			t.Fatal(err)
		}
		clock.now = clock.now.Add(app.OAuthAccessTTL)
		if _, err := exchange(expired); !isOAuthError(err, "invalid_grant") {
			t.Errorf("expired code: %v", err)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.AccessToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("expired access token: %v", err)
		}
		refreshed, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: tok.RefreshToken, ClientID: cimdID})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.app.OAuth.Purge(e.ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, refreshed.AccessToken); err != nil {
			t.Errorf("refreshed token after purge: %v", err)
		}
		clock.now = clock.now.Add(app.OAuthRefreshTTL)
		if _, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: refreshed.RefreshToken, ClientID: cimdID}); !isOAuthError(err, "invalid_grant") {
			t.Errorf("expired refresh token: %v", err)
		}
	})
}

func mustDecide(t *testing.T, e *env, req app.AuthorizationRequest) string {
	t.Helper()
	to, err := e.app.OAuth.Decide(e.ctx, e.alice, req, true, "")
	if err != nil {
		t.Fatal(err)
	}
	return to
}

func isOAuthError(err error, code string) bool {
	var oe *app.OAuthError
	return errors.As(err, &oe) && oe.Code == code
}

// 解除前に発行した未交換のコード
func TestOAuthRevokeDropsPendingCodes(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		req := authRequest(cimdID, "http://127.0.0.1/callback")
		exchange := func(code string) (app.IssuedOAuthToken, error) {
			return e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
				GrantType: "authorization_code", Code: code, RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
			})
		}
		pending := codeFrom(t, mustDecide(t, e, req))
		if _, err := exchange(codeFrom(t, mustDecide(t, e, req))); err != nil {
			t.Fatal(err)
		}
		grants, err := e.app.OAuth.Grants(e.ctx, e.alice)
		if err != nil || len(grants) != 1 {
			t.Fatalf("grants = %+v, %v", grants, err)
		}
		if err := e.app.OAuth.RevokeGrant(e.ctx, e.alice, grants[0].ID); err != nil {
			t.Fatal(err)
		}
		if _, err := exchange(pending); !isOAuthError(err, "invalid_grant") {
			t.Errorf("pending code after revoke: %v", err)
		}
		if grants, _ := e.app.OAuth.Grants(e.ctx, e.alice); len(grants) != 0 {
			t.Errorf("grants after revoke = %+v", grants)
		}
	})
}

// 使用済みリフレッシュトークンの再送で許可ごと失効
func TestOAuthRefreshReuse(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		req := authRequest(cimdID, "http://127.0.0.1/callback")
		tok, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
			GrantType: "authorization_code", Code: codeFrom(t, mustDecide(t, e, req)), RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
		})
		if err != nil {
			t.Fatal(err)
		}
		refresh := func(token string) (app.IssuedOAuthToken, error) {
			return e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: token, ClientID: cimdID})
		}
		stolen, err := refresh(tok.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := refresh(tok.RefreshToken); !isOAuthError(err, "invalid_grant") {
			t.Errorf("reuse: %v", err)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, stolen.AccessToken); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("access token after reuse: %v", err)
		}
		if _, err := refresh(stolen.RefreshToken); !isOAuthError(err, "invalid_grant") {
			t.Errorf("refresh token after reuse: %v", err)
		}
		if grants, _ := e.app.OAuth.Grants(e.ctx, e.alice); len(grants) != 0 {
			t.Errorf("grants after reuse = %+v", grants)
		}
	})
}

// 更新時のscopeはアクセストークンのみ制限
func TestOAuthRefreshScope(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newOAuthEnv(t, s)
		req := authRequest(cimdID, "http://127.0.0.1/callback")
		tok, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{
			GrantType: "authorization_code", Code: codeFrom(t, mustDecide(t, e, req)), RedirectURI: req.RedirectURI, CodeVerifier: verifier, ClientID: cimdID,
		})
		if err != nil {
			t.Fatal(err)
		}
		narrowed, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: tok.RefreshToken, ClientID: cimdID, Scope: "read"})
		if err != nil {
			t.Fatal(err)
		}
		p, err := e.app.Tokens.Authenticate(e.ctx, narrowed.AccessToken)
		if narrowed.Scope != user.ScopeRead || err != nil || p.Scope != user.ScopeRead {
			t.Errorf("narrowed = %s, principal = %+v, %v", narrowed.Scope, p, err)
		}
		full, err := e.app.OAuth.Exchange(e.ctx, app.TokenRequest{GrantType: "refresh_token", RefreshToken: narrowed.RefreshToken, ClientID: cimdID})
		if err != nil {
			t.Fatal(err)
		}
		if p, err := e.app.Tokens.Authenticate(e.ctx, full.AccessToken); full.Scope != user.ScopeWrite || err != nil || p.Scope != user.ScopeWrite {
			t.Errorf("full = %s, principal = %+v, %v", full.Scope, p, err)
		}
	})
}
