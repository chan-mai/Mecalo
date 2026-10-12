package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/adapter/password"
	"github.com/chan-mai/mecalo/backend/internal/adapter/system"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

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
