// Streamable HTTPのMCPサーバ
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/chan-mai/mecalo/backend/internal/adapter/oauthserver"
	"github.com/chan-mai/mecalo/backend/internal/adapter/tools"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

const Path = app.MCPPath

const instructions = `Mecalo is a meal and nutrition log. Dates are YYYY-MM-DD in the user's time zone; get_day returns today's date.
To record a meal, find the food with search_foods, then call add_meal with its id, a meal slot and an amount in g or one of the food's units.
Nutrient amounts are absent when unknown; coverage below 1 means only part of the weight has a known value.`

type Options struct {
	Version string
	Logger  *slog.Logger
	AllowedOrigins []string
}

func NewHandler(a *app.App, opt Options) http.Handler {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	impl := &mcp.Implementation{Name: "mecalo", Version: opt.Version}
	servers := map[user.Scope]*mcp.Server{}
	for _, scope := range []user.Scope{user.ScopeRead, user.ScopeWrite} {
		s := mcp.NewServer(impl, &mcp.ServerOptions{Instructions: instructions})
		for _, t := range tools.For(scope) {
			s.AddTool(toMCPTool(t), handler(a, t, opt.Logger))
		}
		servers[scope] = s
	}

	streamable := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return servers[scopeOf(auth.TokenInfoFromContext(r.Context()))]
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, Logger: opt.Logger})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && slices.Contains(opt.AllowedOrigins, origin) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
			h.Add("Vary", "Origin")
		}
		// prefright 認証除外
		if r.Method == http.MethodOptions {
			if w.Header().Get("Access-Control-Allow-Origin") != "" {
				h := w.Header()
				h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
				h.Set("Access-Control-Max-Age", "600")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !a.Tokens.Enabled() {
			http.NotFound(w, r)
			return
		}
		token, ok := bearer(r)
		if !ok {
			unauthorized(w, a, "")
			return
		}
		p, err := a.Tokens.Authenticate(r.Context(), token)
		if errors.Is(err, app.ErrUnauthorized) {
			unauthorized(w, a, "invalid_token")
			return
		}
		if err != nil {
			opt.Logger.ErrorContext(r.Context(), "mcp authentication failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		info := &auth.TokenInfo{UserID: string(p.User.ID), Scopes: []string{string(p.Scope)}}
		verified := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) { return info, nil }
		auth.RequireBearerToken(verified, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})(streamable).ServeHTTP(w, r)
	})
}

func bearer(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != ""
}

// OAuth有効時は認可サーバの案内付き
func unauthorized(w http.ResponseWriter, a *app.App, errCode string) {
	if a.OAuth.Enabled() {
		w.Header().Set("WWW-Authenticate", oauthserver.Challenge(a, errCode))
	} else {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func scopeOf(info *auth.TokenInfo) user.Scope {
	if info != nil && len(info.Scopes) == 1 && user.Scope(info.Scopes[0]) == user.ScopeWrite {
		return user.ScopeWrite
	}
	return user.ScopeRead
}

func toMCPTool(t *tools.Tool) *mcp.Tool {
	readOnly := t.Scope == user.ScopeRead
	ann := &mcp.ToolAnnotations{ReadOnlyHint: readOnly, OpenWorldHint: new(bool)}
	if !readOnly {
		ann.DestructiveHint = &t.Destructive
	}
	return &mcp.Tool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema, Annotations: ann}
}

func handler(a *app.App, t *tools.Tool, logger *slog.Logger) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := req.Extra.TokenInfo
		if info == nil || info.UserID == "" || !scopeOf(info).Allows(t.Scope) {
			return nil, errors.New("unauthorized")
		}
		out, err := t.Run(ctx, a, user.ID(info.UserID), req.Params.Arguments)
		if err != nil {
			if tools.IsUserError(err) {
				return errorResult(err.Error()), nil
			}
			logger.ErrorContext(ctx, "mcp tool failed", "tool", t.Name, "error", err)
			return errorResult("internal error"), nil
		}
		b, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, StructuredContent: json.RawMessage(b)}, nil
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}
