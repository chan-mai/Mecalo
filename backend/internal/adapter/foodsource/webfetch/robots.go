package webfetch

import (
	"strings"
)

type rule struct {
	allow   bool
	pattern string
}

type rules struct {
	disallowAll bool
	list        []rule
}

// 自身の名前のグループを優先, なければ*のグループ
func parseRobots(body, agent string) rules {
	agent = strings.ToLower(agent)
	type group struct {
		agents []string
		rules  []rule
	}
	var (
		groups []*group
		cur    *group
		inList bool
	)
	for _, line := range strings.Split(body, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "user-agent":
			if cur == nil || !inList {
				cur = &group{}
				groups = append(groups, cur)
			}
			cur.agents = append(cur.agents, strings.ToLower(value))
			inList = true
		case "allow", "disallow":
			inList = false
			if cur == nil {
				continue
			}
			if value == "" {
				continue
			}
			cur.rules = append(cur.rules, rule{allow: key == "allow", pattern: value})
		}
	}
	var star *group
	for _, g := range groups {
		for _, a := range g.agents {
			if a == agent {
				return rules{list: g.rules}
			}
			if a == "*" && star == nil {
				star = g
			}
		}
	}
	if star != nil {
		return rules{list: star.rules}
	}
	return rules{}
}

// 最長一致, 同長はAllow優先
func (r rules) allows(path string) bool {
	if r.disallowAll {
		return false
	}
	best, allow := -1, true
	for _, ru := range r.list {
		if !match(ru.pattern, path) {
			continue
		}
		n := len(ru.pattern)
		if n > best || (n == best && ru.allow) {
			best, allow = n, ru.allow
		}
	}
	return allow
}

// *は任意の文字列, 末尾$は終端
func match(pattern, path string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	pattern = strings.TrimSuffix(pattern, "$")
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	pos := len(parts[0])
	for _, p := range parts[1:] {
		i := strings.Index(path[pos:], p)
		if i < 0 {
			return false
		}
		pos += i + len(p)
	}
	if !anchored {
		return true
	}
	if len(parts) == 1 {
		return pos == len(path)
	}
	return strings.HasSuffix(path, parts[len(parts)-1])
}
