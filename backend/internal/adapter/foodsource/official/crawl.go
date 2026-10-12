package official

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/extract"
	"github.com/chan-mai/mecalo/backend/internal/adapter/foodsource/webfetch"
)

// 一覧ページを辿り商品ページのパスを収集
type crawler struct {
	fetch *webfetch.Client
	base  string
	// 辿る一覧ページのパス
	follow func(path string) bool
	item   *regexp.Regexp
	// 一覧ページ数の上限
	maxPages int
	// 絶対URLで同一サイトとみなすホスト
	hosts []string
}

// 発見順の商品パス
func (c crawler) collect(ctx context.Context, roots []string) ([]string, error) {
	queue := append([]string(nil), roots...)
	visited := map[string]bool{}
	found := map[string]bool{}
	var items []string
	for len(queue) > 0 && len(visited) < c.maxPages {
		path := queue[0]
		queue = queue[1:]
		if visited[path] {
			continue
		}
		visited[path] = true
		res, err := c.fetch.Get(ctx, c.base+path, webfetch.Validators{})
		if errors.Is(err, webfetch.ErrNotFound) || errors.Is(err, webfetch.ErrDisallowed) {
			continue
		}
		if err != nil {
			return nil, err
		}
		doc, err := extract.ParseHTML(res.Body)
		if err != nil {
			return nil, err
		}
		for _, a := range doc.FindAll("a", "") {
			p := c.local(a.Attr("href"))
			switch {
			case p == "":
			case c.item.MatchString(p):
				if !found[p] {
					found[p] = true
					items = append(items, p)
				}
			case c.follow(p) && !visited[p]:
				queue = append(queue, p)
			}
		}
	}
	return items, nil
}

// 同一サイト内リンクのパス, クエリは保持
func (c crawler) local(href string) string {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || (u.Host != "" && !slices.Contains(c.hosts, u.Host)) {
		return ""
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") {
		return ""
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}
