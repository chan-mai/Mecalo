// OpenAPI定義に基づくHTTP API
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi/gen"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

const (
	BasePath      = "/api"
	sessionCookie = "mecalo_session"
	// 状態変更リクエストに必須のヘッダ, クロスサイト送信の抑止
	csrfHeader = "X-Mecalo-Request"
)

type Options struct {
	LocalAuth bool
	// nil: OIDC無効
	OIDC OIDCProvider
	// Secure属性付与, HTTPS配信時は必須
	SecureCookie bool
	// Nuxt以外のクライアント向けCORS許可オリジン
	AllowedOrigins []string
	Logger         *slog.Logger
}

type server struct {
	app *app.App
	opt Options
}

var publicOperations = []string{
	"AuthApiConfig", "AuthApiSetup", "AuthApiSignUp", "AuthApiLogin",
	"AuthApiOidcStart", "AuthApiOidcCallback", "HealthApiGet",
}

func NewHandler(a *app.App, opt Options) (http.Handler, error) {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	s := &server{app: a, opt: opt}

	spec, err := gen.GetSpec()
	if err != nil {
		return nil, err
	}
	validator := middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		Options:               openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
		SilenceServersWarning: true,
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts middleware.ErrorHandlerOpts) {
			writeProblem(w, opts.StatusCode, http.StatusText(opts.StatusCode), err.Error())
		},
	})

	strict := gen.NewStrictHandlerWithOptions(s, []gen.StrictMiddlewareFunc{s.authenticate}, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProblem(w, http.StatusBadRequest, "Bad Request", err.Error())
		},
		ResponseErrorHandlerFunc: s.writeError,
	})
	mux := http.NewServeMux()
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseURL:    BasePath,
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProblem(w, http.StatusBadRequest, "Bad Request", err.Error())
		},
	})

	var h http.Handler = mux
	h = validator(h)
	h = s.csrf(h)
	h = s.cors(h)
	return h, nil
}

type ctxKey int

const (
	userKey ctxKey = iota
	writerKey
	requestKey
)

func currentUser(ctx context.Context) user.User {
	u, _ := ctx.Value(userKey).(user.User)
	return u
}

// Cookie設定用
func responseWriter(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(writerKey).(http.ResponseWriter)
	return w
}

// Cookie参照用
func request(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestKey).(*http.Request)
	return r
}

func (s *server) authenticate(f gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
	public := slices.Contains(publicOperations, operationID)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		ctx = context.WithValue(ctx, writerKey, w)
		ctx = context.WithValue(ctx, requestKey, r)
		if c, err := r.Cookie(sessionCookie); err == nil {
			u, err := s.app.Accounts.Authenticate(ctx, c.Value)
			switch {
			case err == nil:
				ctx = context.WithValue(ctx, userKey, u)
			case !errors.Is(err, app.ErrUnauthorized):
				return nil, err
			}
		}
		if !public && currentUser(ctx).ID == "" {
			return nil, app.ErrUnauthorized
		}
		return f(ctx, w, r, req)
	}
}

func (s *server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(csrfHeader) == "" {
				writeProblem(w, http.StatusForbidden, "Forbidden", "missing "+csrfHeader+" header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && slices.Contains(s.opt.AllowedOrigins, origin) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
				h.Set("Access-Control-Allow-Headers", "Content-Type, "+csrfHeader)
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) setSession(ctx context.Context, sess app.IssuedSession) {
	http.SetCookie(responseWriter(ctx), &http.Cookie{
		Name:     sessionCookie,
		Value:    sess.Token,
		Path:     BasePath,
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   s.opt.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) clearSession(ctx context.Context) {
	http.SetCookie(responseWriter(ctx), &http.Cookie{
		Name:     sessionCookie,
		Path:     BasePath,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.opt.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, app.ErrInvalid):
		writeProblem(w, http.StatusBadRequest, "Bad Request", strings.TrimPrefix(err.Error(), app.ErrInvalid.Error()+": "))
	case errors.Is(err, app.ErrUnauthorized):
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "")
	case errors.Is(err, app.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "Forbidden", "")
	case errors.Is(err, app.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "Not Found", "")
	case errors.Is(err, app.ErrConflict):
		writeProblem(w, http.StatusConflict, "Conflict", "")
	case errors.Is(err, app.ErrDisabled):
		writeProblem(w, http.StatusNotFound, "Not Found", "feature disabled")
	default:
		s.opt.Logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "")
	}
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	p := gen.Problem{Status: int32(status), Title: title}
	if detail != "" {
		p.Detail = &detail
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
