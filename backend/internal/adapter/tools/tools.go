// アシスタントから呼ぶ操作の定義, MCPとWebUIのアシスタントで共有
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

type Tool struct {
	Name        string
	Description string
	// 必要なトークンのスコープ
	Scope user.Scope
	// 削除等の取り消せない操作
	Destructive bool
	InputSchema *jsonschema.Schema
	run         func(ctx context.Context, a *app.App, uid user.ID, args json.RawMessage) (any, error)
	resolved    *jsonschema.Resolved
}

// 引数の不正, アシスタントへそのまま返す
type InputError struct {
	Err error
}

func (e *InputError) Error() string { return e.Err.Error() }
func (e *InputError) Unwrap() error { return e.Err }

// 引数を検証して実行, 結果はJSONへ変換できる値
func (t *Tool) Run(ctx context.Context, a *app.App, uid user.ID, args json.RawMessage) (any, error) {
	if len(bytes.TrimSpace(args)) == 0 || bytes.Equal(bytes.TrimSpace(args), []byte("null")) {
		args = json.RawMessage("{}")
	}
	var instance any
	if err := json.Unmarshal(args, &instance); err != nil {
		return nil, &InputError{err}
	}
	if err := t.resolved.Validate(instance); err != nil {
		return nil, &InputError{err}
	}
	return t.run(ctx, a, uid, args)
}

// スコープで使える操作
func For(scope user.Scope) []*Tool {
	var out []*Tool
	for _, t := range all {
		if scope.Allows(t.Scope) {
			out = append(out, t)
		}
	}
	return out
}

func define[In any](name string, scope user.Scope, desc string, fn func(ctx context.Context, a *app.App, uid user.ID, in In) (any, error), opts ...func(*Tool)) *Tool {
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("tool %s: %v", name, err))
	}
	t := &Tool{Name: name, Description: desc, Scope: scope, InputSchema: schema}
	for _, o := range opts {
		o(t)
	}
	if t.resolved, err = schema.Resolve(nil); err != nil {
		panic(fmt.Sprintf("tool %s: %v", name, err))
	}
	t.run = func(ctx context.Context, a *app.App, uid user.ID, args json.RawMessage) (any, error) {
		var in In
		dec := json.NewDecoder(bytes.NewReader(args))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			return nil, &InputError{err}
		}
		return fn(ctx, a, uid, in)
	}
	return t
}

func destructive(t *Tool) { t.Destructive = true }

// プロパティへの制約の追加
func property(name string, fn func(*jsonschema.Schema)) func(*Tool) {
	return func(t *Tool) {
		p, ok := t.InputSchema.Properties[name]
		if !ok {
			panic(fmt.Sprintf("tool %s: no property %s", t.Name, name))
		}
		fn(p)
	}
}

func enum[T ~string](values ...T) func(*jsonschema.Schema) {
	return func(s *jsonschema.Schema) {
		for _, v := range values {
			s.Enum = append(s.Enum, string(v))
		}
	}
}

func dateFormat(s *jsonschema.Schema) {
	s.Pattern = `^\d{4}-\d{2}-\d{2}$`
}

func positive(s *jsonschema.Schema) {
	zero := 0.0
	s.ExclusiveMinimum = &zero
}

func between(lo, hi float64) func(*jsonschema.Schema) {
	return func(s *jsonschema.Schema) {
		s.Minimum, s.Maximum = &lo, &hi
	}
}

// 利用者の操作で起こりうる失敗か
func IsUserError(err error) bool {
	var ie *InputError
	return errors.As(err, &ie) ||
		errors.Is(err, app.ErrInvalid) || errors.Is(err, app.ErrNotFound) ||
		errors.Is(err, app.ErrForbidden) || errors.Is(err, app.ErrConflict)
}
