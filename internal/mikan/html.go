package mikan

import "golang.org/x/net/html"

// html helper utilities for class/id based traversal.

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range splitClasses(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func splitClasses(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb []byte
	collectText(n, &sb)
	return string(sb)
}

func collectText(n *html.Node, sb *[]byte) {
	if n.Type == html.TextNode {
		*sb = append(*sb, []byte(n.Data)...)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, sb)
	}
}

func nodeOwnText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb []byte
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb = append(sb, []byte(c.Data)...)
		}
	}
	return string(sb)
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func forEachNode(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, fn)
	}
}

// findByClass returns all descendant element nodes with the class.
func findByClass(root *html.Node, class string) []*html.Node {
	var out []*html.Node
	forEachNode(root, func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, class) {
			out = append(out, n)
		}
	})
	return out
}

// firstByClass returns the first descendant with the class.
func firstByClass(root *html.Node, class string) *html.Node {
	var out *html.Node
	forEachNode(root, func(n *html.Node) {
		if out != nil {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, class) {
			out = n
		}
	})
	return out
}

// childByTag returns direct child element nodes with the tag.
func childByTag(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			out = append(out, c)
		}
	}
	return out
}

// firstChild returns the first direct child element with the tag.
func firstChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (tag == "" || c.Data == tag) {
			return c
		}
	}
	return nil
}

// findById returns the first descendant element with the id.
func findById(root *html.Node, id string) *html.Node {
	var out *html.Node
	forEachNode(root, func(n *html.Node) {
		if out != nil {
			return
		}
		if n.Type == html.ElementNode && attr(n, "id") == id {
			out = n
		}
	})
	return out
}

// nextSibling returns the next element sibling.
func nextSibling(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}