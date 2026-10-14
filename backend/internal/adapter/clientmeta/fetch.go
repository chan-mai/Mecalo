// OAuth Client ID Metadata Documentの取得
package clientmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chan-mai/mecalo/backend/internal/port"
)

const (
	maxDocumentSize = 5 << 10
	defaultTTL      = 10 * time.Minute
	minTTL          = time.Minute
	maxTTL          = 24 * time.Hour
	maxEntries      = 256
)

type entry struct {
	meta    port.ClientMetadata
	expires time.Time
}

type Fetcher struct {
	client *http.Client
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]entry
}

// 内部ネットワーク除外
func New(userAgent string) *Fetcher {
	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: denyPrivate}
	return newFetcher(&http.Transport{DialContext: dialer.DialContext, TLSHandshakeTimeout: 5 * time.Second}, userAgent)
}

func newFetcher(rt http.RoundTripper, userAgent string) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: userAgentTransport{rt, userAgent},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now:   time.Now,
		cache: map[string]entry{},
	}
}

type userAgentTransport struct {
	rt http.RoundTripper
	ua string
}

func (t userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("User-Agent", t.ua)
	r.Header.Set("Accept", "application/json")
	return t.rt.RoundTrip(r)
}

func denyPrivate(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return fmt.Errorf("address %s is not allowed", host)
	}
	return nil
}

type document struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (f *Fetcher) Fetch(ctx context.Context, clientID string) (port.ClientMetadata, error) {
	u, err := url.Parse(clientID)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path == "" || u.Path == "/" || u.Fragment != "" || u.User != nil {
		return port.ClientMetadata{}, errors.New("client_id must be an https URL with a path")
	}
	f.mu.Lock()
	e, ok := f.cache[clientID]
	f.mu.Unlock()
	if ok && f.now().Before(e.expires) {
		return e.meta, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return port.ClientMetadata{}, err
	}
	res, err := f.client.Do(req)
	if err != nil {
		return port.ClientMetadata{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return port.ClientMetadata{}, fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxDocumentSize+1))
	if err != nil {
		return port.ClientMetadata{}, err
	}
	if len(body) > maxDocumentSize {
		return port.ClientMetadata{}, errors.New("document too large")
	}
	var doc document
	if err := json.Unmarshal(body, &doc); err != nil {
		return port.ClientMetadata{}, fmt.Errorf("invalid document: %w", err)
	}
	switch {
	case doc.ClientID != clientID:
		return port.ClientMetadata{}, errors.New("client_id in the document does not match the URL")
	case strings.TrimSpace(doc.ClientName) == "":
		return port.ClientMetadata{}, errors.New("client_name is required")
	case len(doc.RedirectURIs) == 0:
		return port.ClientMetadata{}, errors.New("redirect_uris is required")
	case doc.TokenEndpointAuthMethod != "" && doc.TokenEndpointAuthMethod != "none":
		return port.ClientMetadata{}, errors.New("only token_endpoint_auth_method none is supported")
	}
	meta := port.ClientMetadata{ClientID: doc.ClientID, Name: strings.TrimSpace(doc.ClientName), RedirectURIs: doc.RedirectURIs}

	f.mu.Lock()
	if len(f.cache) >= maxEntries {
		clear(f.cache)
	}
	f.cache[clientID] = entry{meta: meta, expires: f.now().Add(ttl(res.Header.Get("Cache-Control")))}
	f.mu.Unlock()
	return meta, nil
}

// max-ageを上下限の範囲で採用
func ttl(cacheControl string) time.Duration {
	for _, d := range strings.Split(cacheControl, ",") {
		d = strings.TrimSpace(strings.ToLower(d))
		if d == "no-store" || d == "no-cache" {
			return minTTL
		}
		if v, ok := strings.CutPrefix(d, "max-age="); ok {
			if n, err := strconv.Atoi(v); err == nil {
				return min(max(time.Duration(n)*time.Second, minTTL), maxTTL)
			}
		}
	}
	return defaultTTL
}
