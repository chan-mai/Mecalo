package mcpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/chan-mai/mecalo/backend/data"
	"github.com/chan-mai/mecalo/backend/internal/adapter/dri"
	"github.com/chan-mai/mecalo/backend/internal/adapter/mcpserver"
	"github.com/chan-mai/mecalo/backend/internal/adapter/oauthserver"
	"github.com/chan-mai/mecalo/backend/internal/adapter/password"
	"github.com/chan-mai/mecalo/backend/internal/adapter/sqlstore"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type env struct {
	ctx   context.Context
	app   *app.App
	url   string
	base  string
	alice user.User
	bob   user.User
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	store, err := sqlstore.OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	ref, err := dri.Load(data.DRI2025)
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(app.Deps{
		Store: store, Clock: system.NewClock(time.UTC), IDs: system.UUIDs{}, Reference: ref, Hasher: password.Fast(),
	}, app.Options{RegistrationOpen: true, MCPEnabled: true})
	e := &env{ctx: ctx, app: a}
	sess, err := a.Accounts.Setup(ctx, "", app.SignUpInput{LoginName: "alice", Password: "password-a"})
	if err != nil {
		t.Fatal(err)
	}
	e.alice = sess.User
	sess, err = a.Accounts.SignUp(ctx, app.SignUpInput{LoginName: "bob", Password: "password-b"})
	if err != nil {
		t.Fatal(err)
	}
	e.bob = sess.User

	mux := http.NewServeMux()
	mux.Handle(mcpserver.Path, mcpserver.NewHandler(a, mcpserver.Options{Version: "test", AllowedOrigins: []string{"https://inspector.example.com"}}))
	oauthserver.Register(mux, a, nil)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	a.Reconfigure(app.Live{Options: app.Options{RegistrationOpen: true, MCPEnabled: true, PublicURL: srv.URL}})
	e.base = srv.URL
	e.url = srv.URL + mcpserver.Path
	return e
}

func (e *env) token(t *testing.T, u user.User, scope user.Scope) app.IssuedToken {
	t.Helper()
	tok, err := e.app.Tokens.Issue(e.ctx, u, "test", scope)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

type bearer string

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+string(b))
	return http.DefaultTransport.RoundTrip(r)
}

func (e *env) connect(t *testing.T, secret string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	cs, err := client.Connect(e.ctx, &mcp.StreamableClientTransport{
		Endpoint: e.url, HTTPClient: &http.Client{Transport: bearer(secret)}, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// ツールの結果, エラー時はメッセージ
func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (map[string]any, string) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err.Error()
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if res.IsError {
		return nil, text
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	return out, ""
}

func postStatus(t *testing.T, url, auth string) int {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return res.StatusCode
}

func TestAuthentication(t *testing.T) {
	e := newEnv(t)
	tok := e.token(t, e.alice, user.ScopeRead)

	if got := postStatus(t, e.url, ""); got != http.StatusUnauthorized {
		t.Errorf("no token: %d", got)
	}
	if got := postStatus(t, e.url, "Bearer mecalo_unknown"); got != http.StatusUnauthorized {
		t.Errorf("unknown token: %d", got)
	}
	if got := postStatus(t, e.url, "Bearer "+tok.Secret); got != http.StatusOK {
		t.Errorf("valid token: %d", got)
	}

	if err := e.app.Tokens.Revoke(e.ctx, e.alice, tok.ID); err != nil {
		t.Fatal(err)
	}
	if got := postStatus(t, e.url, "Bearer "+tok.Secret); got != http.StatusUnauthorized {
		t.Errorf("revoked token: %d", got)
	}

	tok = e.token(t, e.alice, user.ScopeRead)
	e.app.Reconfigure(app.Live{})
	if got := postStatus(t, e.url, "Bearer "+tok.Secret); got != http.StatusNotFound {
		t.Errorf("disabled: %d", got)
	}
}

func TestScope(t *testing.T) {
	e := newEnv(t)
	cs := e.connect(t, e.token(t, e.alice, user.ScopeRead).Secret)

	res, err := cs.ListTools(e.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	if !slices.Contains(names, "search_foods") || slices.Contains(names, "add_meal") || slices.Contains(names, "record_weight") {
		t.Errorf("read tools = %v", names)
	}
	if _, msg := call(t, cs, "record_weight", map[string]any{"kg": 60}); msg == "" {
		t.Error("write tool allowed with read token")
	}
	if ws, err := e.app.Weights.Range(e.ctx, e.alice.ID, e.app.Today(), e.app.Today()); err != nil || len(ws) != 0 {
		t.Errorf("weights = %v, %v", ws, err)
	}
}

func TestOwnership(t *testing.T) {
	e := newEnv(t)
	private, err := e.app.Foods.Create(e.ctx, e.alice.ID, app.FoodInput{
		Name:      "自家製グラノーラ",
		Nutrients: nutrient.Profile{nutrient.Energy: nutrient.Measured(decimal.NewFromInt(450))},
	})
	if err != nil {
		t.Fatal(err)
	}
	alice := e.connect(t, e.token(t, e.alice, user.ScopeWrite).Secret)
	bob := e.connect(t, e.token(t, e.bob, user.ScopeWrite).Secret)

	entry, msg := call(t, alice, "add_meal", map[string]any{"slot": "breakfast", "food_id": string(private.Food.ID), "amount": 50})
	if msg != "" {
		t.Fatal(msg)
	}
	day, msg := call(t, alice, "get_day", nil)
	if msg != "" {
		t.Fatal(msg)
	}
	if n := len(day["entries"].([]any)); n != 1 {
		t.Errorf("alice entries = %d", n)
	}
	energy := day["total"].(map[string]any)["energy"].(map[string]any)["amount"]
	if energy != 225.0 {
		t.Errorf("energy = %v", energy)
	}

	day, _ = call(t, bob, "get_day", nil)
	if n := len(day["entries"].([]any)); n != 0 {
		t.Errorf("bob sees %d entries", n)
	}
	found, _ := call(t, bob, "search_foods", map[string]any{"query": "グラノーラ"})
	if n := len(found["foods"].([]any)); n != 0 {
		t.Errorf("bob finds %d private foods", n)
	}
	if _, msg := call(t, bob, "get_food", map[string]any{"id": string(private.Food.ID)}); msg == "" {
		t.Error("bob read alice's food")
	}
	if _, msg := call(t, bob, "add_meal", map[string]any{"slot": "lunch", "food_id": string(private.Food.ID), "amount": 50}); msg == "" {
		t.Error("bob recorded alice's food")
	}
	id := entry["id"].(string)
	if _, msg := call(t, bob, "update_meal", map[string]any{"id": id, "amount": 100}); msg == "" {
		t.Error("bob updated alice's meal")
	}
	if _, msg := call(t, bob, "delete_meal", map[string]any{"id": id}); msg == "" {
		t.Error("bob deleted alice's meal")
	}
	got, err := e.app.Meals.Day(e.ctx, e.alice.ID, e.app.Today())
	if err != nil || len(got) != 1 || !got[0].Quantity.Amount.Equal(decimal.NewFromInt(50)) {
		t.Errorf("alice entries after bob = %+v, %v", got, err)
	}
}

func TestInputValidation(t *testing.T) {
	e := newEnv(t)
	cs := e.connect(t, e.token(t, e.alice, user.ScopeWrite).Secret)
	cases := []struct {
		name string
		args map[string]any
	}{
		{"add_meal", map[string]any{"slot": "brunch", "food_id": "x", "amount": 1}},
		{"add_meal", map[string]any{"slot": "lunch", "food_id": "x", "amount": -1}},
		{"add_meal", map[string]any{"slot": "lunch", "amount": 1}},
		{"get_day", map[string]any{"date": "2026/10/13"}},
		{"get_day", map[string]any{"extra": true}},
		{"get_range", map[string]any{"from": "2026-01-01", "to": "2026-03-01"}},
		{"record_weight", map[string]any{"kg": 60, "body_fat_pct": 120}},
	}
	for _, c := range cases {
		if _, msg := call(t, cs, c.name, c.args); msg == "" {
			t.Errorf("%s %v accepted", c.name, c.args)
		}
	}
	w, msg := call(t, cs, "record_weight", map[string]any{"date": "2026-10-01", "kg": 60.5})
	if msg != "" || w["kg"] != 60.5 {
		t.Errorf("record_weight = %v, %s", w, msg)
	}
	r, msg := call(t, cs, "get_range", map[string]any{"from": "2026-09-25", "to": "2026-10-01"})
	if msg != "" || len(r["days"].([]any)) != 7 || len(r["weights"].([]any)) != 1 {
		t.Errorf("get_range = %v, %s", r, msg)
	}
}
