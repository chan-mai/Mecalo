package webfetch

import "testing"

func TestRobots(t *testing.T) {
	body := `
User-agent: ClaudeBot
User-agent: GPTBot
Disallow: /

UserAgent: *
User-agent: *
Disallow: /api/
Allow: /api/public
Disallow: /*.csv$
Disallaw: /pdf/

User-agent: Mecalo-test
Disallow: /private
`
	r := parseRobots(body, "Mecalo")
	tests := map[string]bool{
		"/":               true,
		"/items/1":        true,
		"/api/x":          false,
		"/api/public/x":   true,
		"/data/list.csv":  false,
		"/data/list.csv?": true,
		"/pdf/a.pdf":      true,
		"/private":        true,
	}
	for path, want := range tests {
		if got := r.allows(path); got != want {
			t.Errorf("allows(%q) = %v, want %v", path, got, want)
		}
	}

	own := parseRobots("User-agent: *\nDisallow: /\n\nUser-agent: mecalo\nDisallow: /x\n", "Mecalo")
	if !own.allows("/a") || own.allows("/x/1") {
		t.Error("own group should take precedence over *")
	}
	if (rules{disallowAll: true}).allows("/") {
		t.Error("disallowAll should block")
	}
}

func TestMatchAnchored(t *testing.T) {
	if match("/a$", "/a/a") {
		t.Error("/a$ should not match /a/a")
	}
	if !match("/a$", "/a") {
		t.Error("/a$ should match /a")
	}
}
