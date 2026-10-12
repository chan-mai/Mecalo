package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi/gen"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

type OIDCProvider interface {
	Name() string
	// 認可URL, callback検証用
	Begin(ctx context.Context) (string, OIDCFlow, error)
	Finish(ctx context.Context, code string, flow OIDCFlow) (app.ExternalIdentity, error)
}

type OIDCFlow struct {
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	ReturnTo string `json:"returnTo"`
}

const (
	oidcCookie     = "mecalo_oidc"
	oidcCookiePath = BasePath + "/auth/oidc"
	oidcFlowMaxAge = 600
	loginErrorPath = "/login?error=oidc"
)

func (s *server) AuthApiOidcStart(ctx context.Context, req gen.AuthApiOidcStartRequestObject) (gen.AuthApiOidcStartResponseObject, error) {
	p := s.opt.Auth().OIDC
	if p == nil {
		return nil, app.ErrDisabled
	}
	url, flow, err := p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	flow.ReturnTo = safeReturnTo(str(req.Params.ReturnTo))
	raw, err := json.Marshal(flow)
	if err != nil {
		return nil, err
	}
	http.SetCookie(responseWriter(ctx), &http.Cookie{
		Name:     oidcCookie,
		Value:    base64.RawURLEncoding.EncodeToString(raw),
		Path:     oidcCookiePath,
		MaxAge:   oidcFlowMaxAge,
		HttpOnly: true,
		Secure:   s.opt.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	return gen.AuthApiOidcStart302Response{Headers: gen.AuthApiOidcStart302ResponseHeaders{Location: url}}, nil
}

// 失敗時もブラウザ遷移のためログイン画面へリダイレクト
func (s *server) AuthApiOidcCallback(ctx context.Context, req gen.AuthApiOidcCallbackRequestObject) (gen.AuthApiOidcCallbackResponseObject, error) {
	redirect := func(to string) gen.AuthApiOidcCallbackResponseObject {
		return gen.AuthApiOidcCallback302Response{Headers: gen.AuthApiOidcCallback302ResponseHeaders{Location: to}}
	}
	p := s.opt.Auth().OIDC
	if p == nil {
		return nil, app.ErrDisabled
	}
	http.SetCookie(responseWriter(ctx), &http.Cookie{Name: oidcCookie, Path: oidcCookiePath, MaxAge: -1, HttpOnly: true, Secure: s.opt.SecureCookie})

	flow, err := readFlow(ctx)
	if err != nil || req.Params.Error != nil || req.Params.Code == nil || str(req.Params.State) != flow.State {
		s.opt.Logger.WarnContext(ctx, "oidc callback rejected", "error", err, "provider_error", str(req.Params.Error))
		return redirect(loginErrorPath), nil
	}
	id, err := p.Finish(ctx, *req.Params.Code, flow)
	if err != nil {
		s.opt.Logger.WarnContext(ctx, "oidc login failed", "error", err)
		return redirect(loginErrorPath), nil
	}
	sess, err := s.app.Accounts.LoginExternal(ctx, id)
	if errors.Is(err, app.ErrForbidden) {
		return redirect("/login?error=not_registered"), nil
	}
	if err != nil {
		return nil, err
	}
	s.setSession(ctx, sess)
	return redirect(flow.ReturnTo), nil
}

func readFlow(ctx context.Context) (OIDCFlow, error) {
	c, err := request(ctx).Cookie(oidcCookie)
	if err != nil {
		return OIDCFlow{}, err
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return OIDCFlow{}, err
	}
	var flow OIDCFlow
	if err := json.Unmarshal(raw, &flow); err != nil {
		return OIDCFlow{}, err
	}
	if flow.State == "" {
		return OIDCFlow{}, errors.New("empty oidc state")
	}
	return flow, nil
}

// オープンリダイレクト防止, 同一オリジンの絶対パスのみ許可
func safeReturnTo(p string) string {
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.HasPrefix(p, "/\\") {
		return "/"
	}
	return p
}
