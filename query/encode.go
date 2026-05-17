package query

import (
	"net/url"
	"strings"
)

// encoder builds a Strapi/qs-style bracket-encoded query string. Pairs are
// emitted in insertion order; nested key paths render as key[a][b]=value.
// It is intentionally unexported — callers use the public Query type.
type encoder struct {
	pairs []encodedPair
}

type encodedPair struct {
	path  []string
	value string
}

func (e *encoder) add(path []string, value string) {
	clone := make([]string, len(path))
	copy(clone, path)
	e.pairs = append(e.pairs, encodedPair{path: clone, value: value})
}

func (e *encoder) String() string {
	if len(e.pairs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, p := range e.pairs {
		if i > 0 {
			sb.WriteByte('&')
		}
		writeKey(&sb, p.path)
		sb.WriteByte('=')
		sb.WriteString(url.QueryEscape(p.value))
	}
	return sb.String()
}

func writeKey(sb *strings.Builder, path []string) {
	if len(path) == 0 {
		return
	}
	sb.WriteString(url.QueryEscape(path[0]))
	for _, seg := range path[1:] {
		sb.WriteString(url.QueryEscape("[" + seg + "]"))
	}
}
