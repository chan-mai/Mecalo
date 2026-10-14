// MCP OAuth認可 エンドポイント
package oauthserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

const (
	ProtectedResourcePath   = "/.well-known/oauth-protected-resource"
	AuthorizationServerPath = "/.well-known/oauth-authorization-server"
	RegisterPath            = "/api/oauth/register"
	TokenPath               = "/api/oauth/token"
	// FE同意画面path
	AuthorizePath = "/oauth/authorize"
	maxBodySize   = 64 << 10
)

var scopes = []string{string(user.ScopeRead), string(user.ScopeWrite)}

// MCPエンドポイントの保護リソースメタデータURI
func ResourceMetadataURL(a *app.App) string {
	return a.OAuth.Issuer() + ProtectedResourcePath + app.MCPPath
}

func Register(mux *http.ServeMux, a *app.App, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{app: a, logger: logger}
	mux.Handle(ProtectedResourcePath, cors(h.protectedResource))
	mux.Handle(ProtectedResourcePath+app.MCPPath, cors(h.protectedResource))
	mux.Handle(AuthorizationServerPath, cors(h.authorizationServer))
	mux.Handle(RegisterPath, cors(h.register))
	mux.Handle(TokenPath, cors(h.token))
}

type handler struct {
	app    *app.App
	logger *slog.Logger
}

// 全オリジン許可
func cors(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
			h.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func (h *handler) available(w http.ResponseWriter, r *http.Request, method string) bool {
	if !h.app.OAuth.Enabled() {
		http.NotFound(w, r)
		return false
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// RFC 9728
func (h *handler) protectedResource(w http.ResponseWriter, r *http.Request) {
	if !h.available(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 h.app.OAuth.Resource(),
		"authorization_servers":    []string{h.app.OAuth.Issuer()},
		"scopes_supported":         scopes,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Mecalo",
	})
}

// RFC 8414
func (h *handler) authorizationServer(w http.ResponseWriter, r *http.Request) {
	if !h.available(w, r, http.MethodGet) {
		return
	}
	issuer := h.app.OAuth.Issuer()
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         issuer,
		"authorization_endpoint":                         issuer + AuthorizePath,
		"token_endpoint":                                 issuer + TokenPath,
		"registration_endpoint":                          issuer + RegisterPath,
		"scopes_supported":                               scopes,
		"response_types_supported":                       []string{"code"},
		"response_modes_supported":                       []string{"query"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported":          []string{"none", "client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":               []string{"S256"},
		"client_id_metadata_document_supported":          true,
		"authorization_response_iss_parameter_supported": true,
	})
}

type registration struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// RFC 7591
func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.available(w, r, http.MethodPost) {
		return
	}
	var in registration
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize)).Decode(&in); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "invalid JSON")
		return
	}
	for _, g := range in.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "unsupported grant_type "+g)
			return
		}
	}
	if len(in.ResponseTypes) > 0 && !slices.Equal(in.ResponseTypes, []string{"code"}) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "response_types must be code")
		return
	}
	c, err := h.app.OAuth.Register(r.Context(), app.OAuthRegistration{
		Name: in.ClientName, RedirectURIs: in.RedirectURIs, AuthMethod: in.TokenEndpointAuthMethod,
	})
	if h.failed(w, r, err) {
		return
	}
	out := map[string]any{
		"client_id":                  c.ID,
		"client_id_issued_at":        c.CreatedAt.Unix(),
		"client_name":                c.Name,
		"redirect_uris":              c.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": c.AuthMethod,
	}
	if c.Secret != "" {
		out["client_secret"] = c.Secret
		out["client_secret_expires_at"] = 0
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *handler) token(w http.ResponseWriter, r *http.Request) {
	if !h.available(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	f := r.PostForm
	req := app.TokenRequest{
		GrantType:    f.Get("grant_type"),
		Code:         f.Get("code"),
		RedirectURI:  f.Get("redirect_uri"),
		CodeVerifier: f.Get("code_verifier"),
		RefreshToken: f.Get("refresh_token"),
		ClientID:     f.Get("client_id"),
		ClientSecret: f.Get("client_secret"),
		Resource:     f.Get("resource"),
		Scope:        f.Get("scope"),
	}
	basic := false
	if id, secret, ok := r.BasicAuth(); ok {
		// RFC 6749 2.3.1のフォームエンコード
		id, err1 := url.QueryUnescape(id)
		secret, err2 := url.QueryUnescape(secret)
		if err1 != nil || err2 != nil || (req.ClientID != "" && req.ClientID != id) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid client credentials")
			return
		}
		req.ClientID, req.ClientSecret, basic = id, secret, true
	}
	t, err := h.app.OAuth.Exchange(r.Context(), req)
	var oe *app.OAuthError
	if errors.As(err, &oe) && oe.Code == "invalid_client" {
		if basic {
			w.Header().Set("WWW-Authenticate", `Basic realm="mecalo"`)
		}
		writeOAuthError(w, http.StatusUnauthorized, oe.Code, oe.Description)
		return
	}
	if h.failed(w, r, err) {
		return
	}
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  t.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    int(t.ExpiresIn / time.Second),
		"refresh_token": t.RefreshToken,
		"scope":         scopeString(t.Scope),
	})
}

func scopeString(s user.Scope) string {
	if s == user.ScopeWrite {
		return "read write"
	}
	return string(s)
}

func (h *handler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var oe *app.OAuthError
	switch {
	case errors.As(err, &oe):
		writeOAuthError(w, http.StatusBadRequest, oe.Code, oe.Description)
	case errors.Is(err, app.ErrDisabled):
		http.NotFound(w, r)
	default:
		h.logger.ErrorContext(r.Context(), "oauth request failed", "path", r.URL.Path, "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "internal error")
	}
	return true
}

// WWW-Authenticateの値
func Challenge(a *app.App, errCode string) string {
	v := `Bearer resource_metadata=` + strconv.Quote(ResourceMetadataURL(a)) + `, scope="read write"`
	if errCode != "" {
		v += `, error=` + strconv.Quote(errCode)
	}
	return v
}
