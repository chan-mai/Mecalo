package app_test

import (
	"errors"
	"testing"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestTokens(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)

		if _, err := e.app.Tokens.Issue(e.ctx, e.alice, "claude", user.ScopeRead); !errors.Is(err, app.ErrDisabled) {
			t.Errorf("issue while disabled: %v", err)
		}
		e.app.Reconfigure(app.Live{Options: app.Options{MCPEnabled: true}})

		if _, err := e.app.Tokens.Issue(e.ctx, e.alice, " ", user.ScopeRead); !errors.Is(err, app.ErrInvalid) {
			t.Errorf("empty name: %v", err)
		}
		if _, err := e.app.Tokens.Issue(e.ctx, e.alice, "x", user.Scope("admin")); !errors.Is(err, app.ErrInvalid) {
			t.Errorf("unknown scope: %v", err)
		}
		tok, err := e.app.Tokens.Issue(e.ctx, e.alice, " claude ", user.ScopeWrite)
		if err != nil {
			t.Fatal(err)
		}
		if tok.Name != "claude" || tok.Secret == "" || tok.Hash == tok.Secret {
			t.Errorf("issued = %+v", tok)
		}

		p, err := e.app.Tokens.Authenticate(e.ctx, tok.Secret)
		if err != nil {
			t.Fatal(err)
		}
		if p.User.ID != e.alice.ID || p.Scope != user.ScopeWrite {
			t.Errorf("principal = %+v", p)
		}
		for _, bad := range []string{"", "mecalo_x", tok.Secret + "x"} {
			if _, err := e.app.Tokens.Authenticate(e.ctx, bad); !errors.Is(err, app.ErrUnauthorized) {
				t.Errorf("authenticate %q: %v", bad, err)
			}
		}

		if list, err := e.app.Tokens.List(e.ctx, e.bob); err != nil || len(list) != 0 {
			t.Errorf("bob tokens = %v, %v", list, err)
		}
		if err := e.app.Tokens.Revoke(e.ctx, e.bob, tok.ID); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("revoke by other user: %v", err)
		}
		if err := e.app.Tokens.Revoke(e.ctx, e.alice, tok.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.Secret); !errors.Is(err, app.ErrUnauthorized) {
			t.Errorf("revoked token: %v", err)
		}

		tok, err = e.app.Tokens.Issue(e.ctx, e.alice, "claude", user.ScopeRead)
		if err != nil {
			t.Fatal(err)
		}
		e.app.Reconfigure(app.Live{})
		if _, err := e.app.Tokens.Authenticate(e.ctx, tok.Secret); !errors.Is(err, app.ErrDisabled) {
			t.Errorf("authenticate while disabled: %v", err)
		}
		if list, err := e.app.Tokens.List(e.ctx, e.alice); err != nil || len(list) != 1 {
			t.Errorf("list while disabled = %v, %v", list, err)
		}
	})
}
