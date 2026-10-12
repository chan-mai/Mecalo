// 外部サイト取得の共通処理, robots.txt・間隔制御・条件付きGET
package webfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultInterval = 3 * time.Second
	defaultTimeout  = 120 * time.Second
	defaultRetry    = 10 * time.Second
	// 一時的な失敗時の試行回数
	attempts  = 3
	maxBody   = 32 << 20
	robotsTTL = 24 * time.Hour
)

// 既定のUser-Agent, %sはバージョン
const DefaultUserAgent = "Mecalo-Agent/v%s (https://github.com/chan-mai/Mecalo)"

var (
	ErrDisallowed = errors.New("disallowed by robots.txt")
	ErrNotFound   = errors.New("remote resource not found")
)

type Client struct {
	HTTP      *http.Client
	UserAgent string
	// 同一ホストへの最小間隔
	Interval time.Duration
	// 再試行までの待ち時間, 回数に比例
	RetryDelay time.Duration

	mu     sync.Mutex
	next   map[string]time.Time
	robots map[string]robotsEntry
}

type robotsEntry struct {
	rules   rules
	fetched time.Time
}

func New(userAgent string) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: defaultTimeout},
		UserAgent:  userAgent,
		Interval:   defaultInterval,
		RetryDelay: defaultRetry,
	}
}

// robots.txtのUser-agentとの照合名, User-Agentの製品名部分
func (c *Client) agent() string {
	name, _, _ := strings.Cut(c.UserAgent, "/")
	name, _, _ = strings.Cut(name, " ")
	return name
}

// 接続失敗・タイムアウト・5xx・429
var errTemporary = errors.New("temporary failure")

// 前回取得時の検証子
type Validators struct {
	ETag         string
	LastModified string
}

type Response struct {
	Body        []byte
	Validators  Validators
	NotModified bool
}

// 一時的な失敗は間隔を空けて再試行
func (c *Client) Get(ctx context.Context, rawURL string, prev Validators) (Response, error) {
	return c.retry(ctx, func() (Response, error) { return c.send(ctx, http.MethodGet, rawURL, nil, prev) })
}

// フォーム送信で応答を得るAPI用, 条件付き取得なし
func (c *Client) PostForm(ctx context.Context, rawURL string, form url.Values) (Response, error) {
	return c.retry(ctx, func() (Response, error) {
		return c.send(ctx, http.MethodPost, rawURL, form, Validators{})
	})
}

func (c *Client) retry(ctx context.Context, fn func() (Response, error)) (Response, error) {
	var err error
	for i := range attempts {
		var res Response
		res, err = fn()
		if !errors.Is(err, errTemporary) || i == attempts-1 {
			return res, err
		}
		t := time.NewTimer(c.RetryDelay * time.Duration(i+1))
		select {
		case <-ctx.Done():
			t.Stop()
			return Response{}, ctx.Err()
		case <-t.C:
		}
	}
	return Response{}, err
}

func (c *Client) send(ctx context.Context, method, rawURL string, form url.Values, prev Validators) (Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return Response{}, err
	}
	allowed, err := c.allowed(ctx, u)
	if err != nil {
		return Response{}, err
	}
	if !allowed {
		return Response{}, fmt.Errorf("%w: %s", ErrDisallowed, rawURL)
	}
	var payload io.Reader
	if form != nil {
		payload = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, payload)
	if err != nil {
		return Response{}, err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	if prev.ETag != "" {
		req.Header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		req.Header.Set("If-Modified-Since", prev.LastModified)
	}
	res, err := c.do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, err
		}
		return Response{}, fmt.Errorf("%w: %w", errTemporary, err)
	}
	defer res.Body.Close()
	v := Validators{ETag: res.Header.Get("ETag"), LastModified: res.Header.Get("Last-Modified")}
	switch {
	case res.StatusCode == http.StatusNotModified:
		return Response{Validators: prev, NotModified: true}, nil
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return Response{}, fmt.Errorf("%w: %s", ErrNotFound, rawURL)
	case res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests:
		return Response{}, fmt.Errorf("%w: fetch %s: status %d", errTemporary, rawURL, res.StatusCode)
	case res.StatusCode != http.StatusOK:
		return Response{}, fmt.Errorf("fetch %s: status %d", rawURL, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, err
		}
		return Response{}, fmt.Errorf("%w: %w", errTemporary, err)
	}
	if len(body) > maxBody {
		return Response{}, fmt.Errorf("fetch %s: body too large", rawURL)
	}
	return Response{Body: body, Validators: v}, nil
}

// ホスト単位の間隔を空けて送信
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if err := c.wait(req.Context(), req.URL.Host); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	return c.HTTP.Do(req)
}

func (c *Client) wait(ctx context.Context, host string) error {
	c.mu.Lock()
	if c.next == nil {
		c.next = map[string]time.Time{}
	}
	now := time.Now()
	at := c.next[host]
	if at.Before(now) {
		at = now
	}
	c.next[host] = at.Add(c.Interval)
	c.mu.Unlock()
	if d := at.Sub(now); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return nil
}

func (c *Client) allowed(ctx context.Context, u *url.URL) (bool, error) {
	origin := u.Scheme + "://" + u.Host
	c.mu.Lock()
	entry, ok := c.robots[origin]
	c.mu.Unlock()
	if !ok || time.Since(entry.fetched) > robotsTTL {
		r, err := c.fetchRobots(ctx, origin)
		if err != nil {
			return false, err
		}
		entry = robotsEntry{rules: r, fetched: time.Now()}
		c.mu.Lock()
		if c.robots == nil {
			c.robots = map[string]robotsEntry{}
		}
		c.robots[origin] = entry
		c.mu.Unlock()
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return entry.rules.allows(path), nil
}

// 4xxは全許可, 5xx・接続失敗は全拒否(RFC 9309)
func (c *Client) fetchRobots(ctx context.Context, origin string) (rules, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/robots.txt", nil)
	if err != nil {
		return rules{}, err
	}
	res, err := c.do(req)
	if err != nil {
		return rules{}, fmt.Errorf("fetch robots.txt: %w", err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode >= 500:
		return rules{disallowAll: true}, nil
	case res.StatusCode >= 400:
		return rules{}, nil
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 512<<10))
	if err != nil {
		return rules{}, err
	}
	return parseRobots(string(body), c.agent()), nil
}

// 大きなファイル用, 呼出側でClose
func (c *Client) Open(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	allowed, err := c.allowed(ctx, u)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("%w: %s", ErrDisallowed, rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	// 大容量の取得はクライアント全体のタイムアウト対象外
	stream := *c.HTTP
	stream.Timeout = 0
	if err := c.wait(ctx, u.Host); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := stream.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("fetch %s: status %d", rawURL, res.StatusCode)
	}
	return res.Body, nil
}
