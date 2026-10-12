package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/adapter/password"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

func TestUpdateAccounts(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)

		acc, err := e.app.Accounts.UpdateProfile(e.ctx, e.bob, app.ProfileUpdate{DisplayName: ptr(" Bobby "), LoginName: ptr("Bob2")})
		if err != nil {
			t.Fatal(err)
		}
		if acc.DisplayName != "Bobby" || acc.LoginName != "bob2" || acc.Role != user.RoleMember {
			t.Errorf("profile = %+v", acc)
		}
		if _, err := e.app.Accounts.LoginLocal(e.ctx, "bob2", "password-b"); err != nil {
			t.Errorf("login with new name: %v", err)
		}
		if _, err := e.app.Accounts.UpdateProfile(e.ctx, e.bob, app.ProfileUpdate{LoginName: ptr("alice")}); !errors.Is(err, app.ErrConflict) {
			t.Errorf("duplicate login: %v", err)
		}
		if _, err := e.app.Accounts.UpdateProfile(e.ctx, e.bob, app.ProfileUpdate{LoginName: ptr("x")}); !errors.Is(err, app.ErrInvalid) {
			t.Errorf("invalid login: %v", err)
		}

		if _, err := e.app.Accounts.UpdateUser(e.ctx, e.bob, e.alice.ID, app.UserUpdate{Role: ptr(user.RoleMember)}); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("member update: %v", err)
		}
		if _, err := e.app.Accounts.UpdateUser(e.ctx, e.alice, e.alice.ID, app.UserUpdate{Role: ptr(user.RoleMember)}); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("self demotion: %v", err)
		}

		sess, err := e.app.Accounts.LoginLocal(e.ctx, "bob2", "password-b")
		if err != nil {
			t.Fatal(err)
		}
		acc, err = e.app.Accounts.UpdateUser(e.ctx, e.alice, e.bob.ID, app.UserUpdate{Role: ptr(user.RoleAdmin), Password: ptr("password-new")})
		if err != nil {
			t.Fatal(err)
		}
		if acc.Role != user.RoleAdmin {
			t.Errorf("role = %+v", acc)
		}
		if _, err := e.app.Accounts.Authenticate(e.ctx, sess.Token); err == nil {
			t.Error("session kept after password reset")
		}
		if _, err := e.app.Accounts.LoginLocal(e.ctx, "bob2", "password-new"); err != nil {
			t.Errorf("login with reset password: %v", err)
		}

		list, err := e.app.Accounts.ListUsers(e.ctx, e.alice)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 2 || list[0].LoginName != "alice" || list[1].LoginName != "bob2" {
			t.Errorf("list = %+v", list)
		}
	})
}

func TestSetupToken(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		a := app.New(app.Deps{
			Store:      s,
			Clock:      system.NewClock(time.UTC),
			IDs:        system.UUIDs{},
			Hasher:     password.Fast(),
			SetupToken: "token",
		}, app.Options{})
		ctx := context.Background()
		in := app.SignUpInput{LoginName: "admin", Password: "password-a"}
		if _, err := a.Accounts.Setup(ctx, "wrong", in); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("wrong token: %v", err)
		}
		sess, err := a.Accounts.Setup(ctx, "token", in)
		if err != nil {
			t.Fatal(err)
		}
		if !sess.User.IsAdmin() {
			t.Errorf("user = %+v", sess.User)
		}
	})
}
