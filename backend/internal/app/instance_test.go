package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

var instanceDefs = []app.SettingDef{
	{Key: "MECALO_REGISTRATION", Kind: app.SettingChoice, Default: "closed", Options: []string{"closed", "open"}},
	{Key: "MECALO_SYNC_INTERVAL", Kind: app.SettingDuration, Default: "168h"},
	{Key: "MECALO_TABECAL_KEY", Kind: app.SettingSecret},
	{Key: "MECALO_OIDC_SCOPES", Kind: app.SettingList, Default: "openid,profile"},
}

func settingOf(t *testing.T, list []app.Setting, key string) app.Setting {
	t.Helper()
	for _, s := range list {
		if s.Key == key {
			return s
		}
	}
	t.Fatalf("setting %s not found", key)
	return app.Setting{}
}

func TestInstanceSettings(t *testing.T) {
	eachStore(t, func(t *testing.T, s port.Store) {
		e := newEnv(t, s)
		var applied map[string]string
		reject := ""
		apply := func(_ context.Context, v map[string]string) (func(), error) {
			if v["MECALO_REGISTRATION"] == reject {
				return nil, errors.New("rejected")
			}
			return func() { applied = v }, nil
		}
		clock := &fixedClock{now: time.Date(2026, 10, 12, 8, 0, 0, 0, jst)}
		env := map[string]string{"MECALO_SYNC_INTERVAL": "24h", "MECALO_TABECAL_KEY": "from-env"}
		inst := app.NewInstance(s, clock, instanceDefs, env, apply)
		if ignored, err := inst.Load(e.ctx); err != nil || ignored != nil {
			t.Fatal(ignored, err)
		}
		if applied["MECALO_SYNC_INTERVAL"] != "24h" || applied["MECALO_REGISTRATION"] != "closed" {
			t.Errorf("load = %v", applied)
		}

		if _, err := inst.List(e.ctx, e.bob); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("member list: %v", err)
		}
		list, err := inst.Update(e.ctx, e.alice, map[string]string{
			"MECALO_REGISTRATION":  "open",
			"MECALO_SYNC_INTERVAL": "24h",
			"MECALO_TABECAL_KEY":   "from-ui",
			"MECALO_OIDC_SCOPES":   " openid , email ,openid",
		})
		if err != nil {
			t.Fatal(err)
		}
		if st := settingOf(t, list, "MECALO_REGISTRATION"); st.Origin != app.OriginUI || st.Value != "open" {
			t.Errorf("registration = %+v", st)
		}
		// 環境変数と同じ値はWebUIでの設定なし
		if st := settingOf(t, list, "MECALO_SYNC_INTERVAL"); st.Origin != app.OriginEnv || st.Value != "24h" {
			t.Errorf("interval = %+v", st)
		}
		if st := settingOf(t, list, "MECALO_TABECAL_KEY"); st.Origin != app.OriginUI || st.Value != "" || st.EnvValue != "" || !st.Configured {
			t.Errorf("secret = %+v", st)
		}
		if applied["MECALO_TABECAL_KEY"] != "from-ui" || applied["MECALO_OIDC_SCOPES"] != "openid,email" {
			t.Errorf("applied = %v", applied)
		}

		// 反映できない値は保存対象外
		reject = "closed"
		if _, err := inst.Update(e.ctx, e.alice, map[string]string{"MECALO_REGISTRATION": "closed"}); !errors.Is(err, app.ErrInvalid) {
			t.Errorf("rejected update: %v", err)
		}
		reject = ""
		for _, bad := range []map[string]string{
			{"MECALO_REGISTRATION": "invite"},
			{"MECALO_SYNC_INTERVAL": "weekly"},
			{"MECALO_UNKNOWN": "x"},
		} {
			if _, err := inst.Update(e.ctx, e.alice, bad); !errors.Is(err, app.ErrInvalid) {
				t.Errorf("update %v: %v", bad, err)
			}
		}
		list, err = inst.List(e.ctx, e.alice)
		if err != nil {
			t.Fatal(err)
		}
		if st := settingOf(t, list, "MECALO_REGISTRATION"); st.Value != "open" {
			t.Errorf("after rejected = %+v", st)
		}

		list, err = inst.Reset(e.ctx, e.alice, []string{"MECALO_TABECAL_KEY"})
		if err != nil {
			t.Fatal(err)
		}
		if st := settingOf(t, list, "MECALO_TABECAL_KEY"); st.Origin != app.OriginEnv || applied["MECALO_TABECAL_KEY"] != "from-env" {
			t.Errorf("reset one = %+v, %v", st, applied)
		}
		if st := settingOf(t, list, "MECALO_REGISTRATION"); st.Origin != app.OriginUI {
			t.Errorf("reset kept = %+v", st)
		}

		// 保存値で反映できない場合は環境変数・既定値
		reject = "open"
		inst = app.NewInstance(s, clock, instanceDefs, env, apply)
		if ignored, err := inst.Load(e.ctx); err != nil || ignored == nil || applied["MECALO_REGISTRATION"] != "closed" {
			t.Errorf("fallback load: %v, %v, %v", ignored, err, applied)
		}
		reject = ""

		list, err = inst.Reset(e.ctx, e.alice, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, st := range list {
			if st.Origin == app.OriginUI {
				t.Errorf("reset all = %+v", st)
			}
		}
		saved, err := s.Instance().List(e.ctx)
		if err != nil || len(saved) != 0 {
			t.Errorf("saved = %v, %v", saved, err)
		}
	})
}
