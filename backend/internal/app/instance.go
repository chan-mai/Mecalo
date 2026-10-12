package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/domain/user"
	"github.com/chan-mai/mecalo/backend/internal/port"
)

type SettingKind string

const (
	SettingText     SettingKind = "text"
	SettingSecret   SettingKind = "secret"
	SettingBool     SettingKind = "bool"
	SettingDuration SettingKind = "duration"
	// Optionsから1つ
	SettingChoice SettingKind = "choice"
	// Optionsから複数, カンマ区切り
	SettingChoices SettingKind = "choices"
	// 任意の値のカンマ区切り
	SettingList SettingKind = "list"
)

type SettingOrigin string

const (
	OriginDefault SettingOrigin = "default"
	OriginEnv     SettingOrigin = "env"
	OriginUI      SettingOrigin = "ui"
)

type SettingDef struct {
	// 環境変数名
	Key     string
	Group   string
	Kind    SettingKind
	Default string
	Options []string
}

type Setting struct {
	SettingDef
	// 秘密の値は空
	Value string
	// 秘密の値は空
	EnvValue string
	HasEnv   bool
	// 値の有無
	Configured bool
	Origin     SettingOrigin
}

// 値の検証と反映の準備, commitで反映
type ApplyFunc func(ctx context.Context, values map[string]string) (commit func(), err error)

// 管理者がWebUIから変更する設定, 値はWebUI > 環境変数 > 既定値の順
type Instance struct {
	store port.Store
	defs  []SettingDef
	env   map[string]string
	apply ApplyFunc
	mu    sync.Mutex
}

func NewInstance(store port.Store, defs []SettingDef, env map[string]string, apply ApplyFunc) *Instance {
	return &Instance{store: store, defs: defs, env: env, apply: apply}
}

// 起動時の反映, 保存値で失敗時は保存値を除き反映
func (s *Instance) Load(ctx context.Context) (ignored, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ui, err := s.store.Instance().List(ctx)
	if err != nil {
		return nil, err
	}
	commit, err := s.apply(ctx, s.effective(ui))
	if err != nil && len(ui) > 0 {
		ignored = err
		commit, err = s.apply(ctx, s.effective(nil))
	}
	if err != nil {
		return ignored, err
	}
	commit()
	return ignored, nil
}

func (s *Instance) List(ctx context.Context, actor user.User) ([]Setting, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	ui, err := s.store.Instance().List(ctx)
	if err != nil {
		return nil, err
	}
	return s.view(ui), nil
}

// 環境変数・既定値と同じ値はWebUIでの設定を解除
func (s *Instance) Update(ctx context.Context, actor user.User, values map[string]string) ([]Setting, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ui, err := s.store.Instance().List(ctx)
	if err != nil {
		return nil, err
	}
	next := maps.Clone(ui)
	put := map[string]string{}
	var del []string
	for k, v := range values {
		d, ok := s.def(k)
		if !ok {
			return nil, invalid(fmt.Errorf("unknown setting %q", k))
		}
		v, err := normalizeSetting(d, v)
		if err != nil {
			return nil, invalid(fmt.Errorf("%s: %w", k, err))
		}
		if v == s.fallback(d) {
			if _, ok := ui[k]; ok {
				del = append(del, k)
			}
			delete(next, k)
			continue
		}
		next[k] = v
		put[k] = v
	}
	return s.save(ctx, next, put, del, false)
}

// keysが空なら全件を環境変数・既定値へ戻す
func (s *Instance) Reset(ctx context.Context, actor user.User, keys []string) ([]Setting, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		if _, ok := s.def(k); !ok {
			return nil, invalid(fmt.Errorf("unknown setting %q", k))
		}
	}
	ui, err := s.store.Instance().List(ctx)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return s.save(ctx, map[string]string{}, nil, nil, true)
	}
	next := maps.Clone(ui)
	for _, k := range keys {
		delete(next, k)
	}
	return s.save(ctx, next, nil, keys, false)
}

// 反映できない値は保存対象外
func (s *Instance) save(ctx context.Context, next, put map[string]string, del []string, all bool) ([]Setting, error) {
	commit, err := s.apply(ctx, s.effective(next))
	if err != nil {
		return nil, invalid(err)
	}
	err = s.store.InTx(ctx, func(r port.Repositories) error {
		if len(put) > 0 {
			if err := r.Instance().Put(ctx, put); err != nil {
				return err
			}
		}
		if all || len(del) > 0 {
			return r.Instance().Delete(ctx, del)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	commit()
	return s.view(next), nil
}

func (s *Instance) def(key string) (SettingDef, bool) {
	i := slices.IndexFunc(s.defs, func(d SettingDef) bool { return d.Key == key })
	if i < 0 {
		return SettingDef{}, false
	}
	return s.defs[i], true
}

func (s *Instance) fallback(d SettingDef) string {
	if v, ok := s.env[d.Key]; ok {
		return v
	}
	return d.Default
}

func (s *Instance) effective(ui map[string]string) map[string]string {
	out := make(map[string]string, len(s.defs))
	for _, d := range s.defs {
		if v, ok := ui[d.Key]; ok {
			out[d.Key] = v
		} else {
			out[d.Key] = s.fallback(d)
		}
	}
	return out
}

func (s *Instance) view(ui map[string]string) []Setting {
	out := make([]Setting, 0, len(s.defs))
	for _, d := range s.defs {
		st := Setting{SettingDef: d, Origin: OriginDefault, Value: d.Default}
		st.EnvValue, st.HasEnv = s.env[d.Key]
		if st.HasEnv {
			st.Origin, st.Value = OriginEnv, st.EnvValue
		}
		if v, ok := ui[d.Key]; ok {
			st.Origin, st.Value = OriginUI, v
		}
		st.Configured = st.Value != ""
		if d.Kind == SettingSecret {
			st.Value, st.EnvValue = "", ""
		}
		out = append(out, st)
	}
	return out
}

// 種類ごとの形式の確認と空白除去
func normalizeSetting(d SettingDef, v string) (string, error) {
	v = strings.TrimSpace(v)
	switch d.Kind {
	case SettingBool:
		if _, err := strconv.ParseBool(v); err != nil {
			return "", fmt.Errorf("invalid boolean %q", v)
		}
	case SettingDuration:
		if _, err := time.ParseDuration(v); err != nil {
			return "", fmt.Errorf("invalid duration %q", v)
		}
	case SettingChoice:
		if !slices.Contains(d.Options, v) {
			return "", fmt.Errorf("unknown value %q", v)
		}
	case SettingChoices, SettingList:
		var items []string
		for item := range strings.SplitSeq(v, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if d.Kind == SettingChoices && !slices.Contains(d.Options, item) {
				return "", fmt.Errorf("unknown value %q", item)
			}
			if !slices.Contains(items, item) {
				items = append(items, item)
			}
		}
		v = strings.Join(items, ",")
	}
	return v, nil
}
