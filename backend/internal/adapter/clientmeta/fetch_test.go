package clientmeta

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetch(t *testing.T) {
	var hits atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		id := srv.URL + r.URL.Path
		switch r.URL.Path {
		case "/ok.json":
			w.Header().Set("Cache-Control", "max-age=3600")
			fmt.Fprintf(w, `{"client_id":%q,"client_name":"Client","redirect_uris":["http://127.0.0.1/cb"],"token_endpoint_auth_method":"none"}`, id)
		case "/mismatch.json":
			fmt.Fprint(w, `{"client_id":"https://other.example.com/x.json","client_name":"Client","redirect_uris":["http://127.0.0.1/cb"]}`)
		case "/noname.json":
			fmt.Fprintf(w, `{"client_id":%q,"redirect_uris":["http://127.0.0.1/cb"]}`, id)
		case "/jwt.json":
			fmt.Fprintf(w, `{"client_id":%q,"client_name":"Client","redirect_uris":["http://127.0.0.1/cb"],"token_endpoint_auth_method":"private_key_jwt"}`, id)
		case "/large.json":
			fmt.Fprint(w, `{"client_name":"`+strings.Repeat("x", maxDocumentSize)+`"}`)
		case "/redirect.json":
			http.Redirect(w, r, "/ok.json", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f := newFetcher(srv.Client().Transport, "test")
	now := time.Now()
	f.now = func() time.Time { return now }
	ctx := context.Background()

	m, err := f.Fetch(ctx, srv.URL+"/ok.json")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Client" || len(m.RedirectURIs) != 1 {
		t.Errorf("meta = %+v", m)
	}
	if _, err := f.Fetch(ctx, srv.URL+"/ok.json"); err != nil || hits.Load() != 1 {
		t.Errorf("cached fetch: hits = %d, %v", hits.Load(), err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := f.Fetch(ctx, srv.URL+"/ok.json"); err != nil || hits.Load() != 2 {
		t.Errorf("expired cache: hits = %d, %v", hits.Load(), err)
	}

	for _, path := range []string{"/mismatch.json", "/noname.json", "/jwt.json", "/large.json", "/redirect.json", "/missing.json"} {
		if _, err := f.Fetch(ctx, srv.URL+path); err == nil {
			t.Errorf("%s accepted", path)
		}
	}
	for _, id := range []string{"http://example.com/c.json", "https://example.com", "https://example.com/", "https://u:p@example.com/c.json"} {
		if _, err := f.Fetch(ctx, id); err == nil {
			t.Errorf("%s accepted", id)
		}
	}
}

func TestDenyPrivate(t *testing.T) {
	for addr, allowed := range map[string]bool{
		"127.0.0.1:443": false, "[::1]:443": false, "10.0.0.1:443": false, "192.168.1.1:443": false,
		"169.254.169.254:80": false, "0.0.0.0:443": false, "[fd00::1]:443": false, "93.184.215.14:443": true,
	} {
		if err := denyPrivate("tcp", addr, nil); (err == nil) != allowed {
			t.Errorf("%s: %v", addr, err)
		}
	}
}

func TestTTL(t *testing.T) {
	cases := map[string]time.Duration{
		"": defaultTTL, "max-age=5": minTTL, "public, max-age=600": 10 * time.Minute, "max-age=999999": maxTTL, "no-store": minTTL,
	}
	for in, want := range cases {
		if got := ttl(in); got != want {
			t.Errorf("ttl(%q) = %v", in, got)
		}
	}
}
