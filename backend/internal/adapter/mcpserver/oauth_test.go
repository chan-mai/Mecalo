package mcpserver_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

func getJSON(t *testing.T, u string, out any) {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d", u, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func postForm(t *testing.T, u string, form url.Values) (int, map[string]any) {
	t.Helper()
	res, err := http.PostForm(u, form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// 発見から/mcpの呼び出しまで
func TestOAuthFlow(t *testing.T) {
	e := newEnv(t)

	res, err := http.Post(e.url, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	challenges, err := oauthex.ParseWWWAuthenticate(res.Header.Values("WWW-Authenticate"))
	if res.StatusCode != http.StatusUnauthorized || err != nil || len(challenges) != 1 {
		t.Fatalf("challenge = %d %v %v", res.StatusCode, res.Header.Values("WWW-Authenticate"), err)
	}
	metaURL := challenges[0].Params["resource_metadata"]

	var prm oauthex.ProtectedResourceMetadata
	getJSON(t, metaURL, &prm)
	if prm.Resource != e.url || len(prm.AuthorizationServers) != 1 || prm.AuthorizationServers[0] != e.base {
		t.Fatalf("protected resource = %+v", prm)
	}
	var asm map[string]any
	getJSON(t, e.base+"/.well-known/oauth-authorization-server", &asm)
	if asm["issuer"] != e.base || asm["authorization_endpoint"] != e.base+"/oauth/authorize" || asm["client_id_metadata_document_supported"] != true {
		t.Fatalf("authorization server = %v", asm)
	}

	redirect := "http://127.0.0.1:33418/callback"
	body := `{"client_name":"Test Client","redirect_uris":["` + redirect + `"],"token_endpoint_auth_method":"none","grant_types":["authorization_code","refresh_token"]}`
	res, err = http.Post(asm["registration_endpoint"].(string), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var reg map[string]any
	_ = json.NewDecoder(res.Body).Decode(&reg)
	res.Body.Close()
	clientID, _ := reg["client_id"].(string)
	if res.StatusCode != http.StatusCreated || clientID == "" || reg["client_secret"] != nil {
		t.Fatalf("register = %d %v", res.StatusCode, reg)
	}

	verifier := strings.Repeat("v", 50)
	sum := sha256.Sum256([]byte(verifier))
	req := app.AuthorizationRequest{
		ResponseType: "code", ClientID: clientID, RedirectURI: redirect, Scope: "read",
		CodeChallenge: base64.RawURLEncoding.EncodeToString(sum[:]), CodeChallengeMethod: "S256", Resource: e.url,
	}
	to, err := e.app.OAuth.Decide(e.ctx, e.alice, req, true, "")
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := url.Parse(to)
	tokenURL := asm["token_endpoint"].(string)
	status, tok := postForm(t, tokenURL, url.Values{
		"grant_type": {"authorization_code"}, "code": {cb.Query().Get("code")}, "redirect_uri": {redirect},
		"code_verifier": {verifier}, "client_id": {clientID}, "resource": {e.url},
	})
	access, _ := tok["access_token"].(string)
	if status != http.StatusOK || access == "" || tok["token_type"] != "Bearer" || tok["scope"] != "read" {
		t.Fatalf("token = %d %v", status, tok)
	}

	cs := e.connect(t, access)
	tools, err := cs.ListTools(e.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 4 {
		t.Errorf("read tools = %d", len(tools.Tools))
	}

	status, refreshed := postForm(t, tokenURL, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tok["refresh_token"].(string)}, "client_id": {clientID}})
	if status != http.StatusOK || refreshed["access_token"] == access {
		t.Fatalf("refresh = %d %v", status, refreshed)
	}
	status, failed := postForm(t, tokenURL, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tok["refresh_token"].(string)}, "client_id": {clientID}})
	if status != http.StatusBadRequest || failed["error"] != "invalid_grant" {
		t.Errorf("refresh reuse = %d %v", status, failed)
	}
	status, failed = postForm(t, tokenURL, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"}, "client_id": {"unknown"}})
	if status != http.StatusUnauthorized || failed["error"] != "invalid_client" {
		t.Errorf("unknown client = %d %v", status, failed)
	}
	if got := postStatus(t, e.url, "Bearer "+refreshed["access_token"].(string)); got != http.StatusUnauthorized {
		t.Errorf("access token after refresh reuse: %d", got)
	}

	// 公開URL未設定時のOAuth無効化
	e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true}})
	res, err = http.Get(metaURL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("metadata without public URL: %d", res.StatusCode)
	}
	if got := postStatus(t, e.url, "Bearer "+e.token(t, e.alice, user.ScopeRead).Secret); got != http.StatusOK {
		t.Errorf("api token without public URL: %d", got)
	}
}

func TestCORS(t *testing.T) {
	e := newEnv(t)
	send := func(method, origin string) (int, http.Header) {
		t.Helper()
		req, err := http.NewRequest(method, e.url, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode, res.Header
	}

	status, h := send(http.MethodOptions, "https://inspector.example.com")
	if status != http.StatusNoContent || h.Get("Access-Control-Allow-Origin") != "https://inspector.example.com" ||
		!strings.Contains(h.Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Errorf("preflight = %d %v", status, h)
	}
	status, h = send(http.MethodPost, "https://inspector.example.com")
	if status != http.StatusUnauthorized || h.Get("Access-Control-Allow-Origin") == "" ||
		!strings.Contains(h.Get("Access-Control-Expose-Headers"), "WWW-Authenticate") {
		t.Errorf("unauthorized = %d %v", status, h)
	}
	if _, h := send(http.MethodOptions, "https://evil.example.com"); h.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin = %v", h)
	}
}
