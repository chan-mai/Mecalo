package catalog

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTML文書の簡易検索
type Node struct {
	*html.Node
}

func ParseHTML(body []byte) (Node, error) {
	n, err := html.Parse(bytes.NewReader(body))
	return Node{n}, err
}

// タグ名(空は任意)とclass(空は任意)で全子孫を検索
func (n Node) FindAll(tag, class string) []Node {
	var out []Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (tag == "" || c.Data == tag) && (class == "" || hasClass(c, class)) {
				out = append(out, Node{c})
			}
			walk(c)
		}
	}
	if n.Node != nil {
		walk(n.Node)
	}
	return out
}

// 最初の一致, なければNode{nil}
func (n Node) Find(tag, class string) Node {
	if all := n.FindAll(tag, class); len(all) > 0 {
		return all[0]
	}
	return Node{}
}

func (n Node) Ok() bool {
	return n.Node != nil
}

func (n Node) Attr(key string) string {
	if n.Node == nil {
		return ""
	}
	for _, a := range n.Node.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// 子孫のテキスト, brは空白, 空白は1つに統一
func (n Node) Text() string {
	if n.Node == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		switch {
		case x.Type == html.TextNode:
			b.WriteString(x.Data)
		case x.Type == html.ElementNode && (x.DataAtom == atom.Br || x.DataAtom == atom.Td || x.DataAtom == atom.Th || x.DataAtom == atom.Li || x.DataAtom == atom.P):
			b.WriteString(" ")
		case x.Type == html.ElementNode && (x.DataAtom == atom.Script || x.DataAtom == atom.Style):
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if x.Type == html.ElementNode && (x.DataAtom == atom.Td || x.DataAtom == atom.Th) {
			b.WriteString(" ")
		}
	}
	walk(n.Node)
	return strings.Join(strings.Fields(b.String()), " ")
}

// dt・ddやth・tdの見出しと値の組
func (n Node) Definitions() map[string]string {
	out := map[string]string{}
	for _, dt := range n.FindAll("dt", "") {
		dd := nextElement(dt.Node)
		if dd != nil && dd.DataAtom == atom.Dd {
			out[strings.TrimSuffix(Normalize(dt.Text()), ":")] = Node{dd}.Text()
		}
	}
	for _, th := range n.FindAll("th", "") {
		td := nextElement(th.Node)
		if td != nil && td.DataAtom == atom.Td {
			key := strings.TrimSuffix(Normalize(th.Text()), ":")
			if _, dup := out[key]; !dup {
				out[key] = Node{td}.Text()
			}
		}
	}
	return out
}

// 行の直下のth・td
func (n Node) Cells() []Node {
	var out []Node
	if n.Node == nil {
		return nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.DataAtom == atom.Th || c.DataAtom == atom.Td) {
			out = append(out, Node{c})
		}
	}
	return out
}

func nextElement(n *html.Node) *html.Node {
	for x := n.NextSibling; x != nil; x = x.NextSibling {
		if x.Type == html.ElementNode {
			return x
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}
