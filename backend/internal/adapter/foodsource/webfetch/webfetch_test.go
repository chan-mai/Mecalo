package webfetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRetry(t *testing.T) {
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		if r.Header.Get("User-Agent") != "Mecalo-Agent/vtest (https://example.com)" {
			t.Errorf("user agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: Mecalo-Agent\nDisallow: /private\n"))
		case "/flaky":
			if calls["/flaky"] == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte("ok"))
		case "/down":
			w.WriteHeader(http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New("Mecalo-Agent/vtest (https://example.com)")
	c.Interval, c.RetryDelay = 0, 0
	ctx := context.Background()

	res, err := c.Get(ctx, srv.URL+"/flaky", Validators{})
	if err != nil || string(res.Body) != "ok" || calls["/flaky"] != 2 {
		t.Errorf("flaky = %q, %v, calls %d", res.Body, err, calls["/flaky"])
	}
	if _, err := c.Get(ctx, srv.URL+"/down", Validators{}); err == nil || calls["/down"] != attempts {
		t.Errorf("down err = %v, calls %d", err, calls["/down"])
	}
	if _, err := c.Get(ctx, srv.URL+"/missing", Validators{}); !errors.Is(err, ErrNotFound) || calls["/missing"] != 1 {
		t.Errorf("missing err = %v, calls %d", err, calls["/missing"])
	}
	// 製品名でrobots.txtのグループを照合
	if _, err := c.Get(ctx, srv.URL+"/private", Validators{}); !errors.Is(err, ErrDisallowed) {
		t.Errorf("private err = %v", err)
	}
}
