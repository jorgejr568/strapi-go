package blocks

import (
	"encoding/json"
	"testing"
)

func TestInlineLinkSatisfiesInlineNode(t *testing.T) {
	// Compile-time sanity: both Text and InlineLink implement InlineNode.
	var _ InlineNode = Text{}
	var _ InlineNode = InlineLink{}
}

func TestDecodeInlinePureText(t *testing.T) {
	raw := []byte(`[
		{"type":"text","text":"hello "},
		{"type":"text","text":"world","bold":true}
	]`)
	got, err := decodeInline(raw)
	if err != nil {
		t.Fatalf("decodeInline: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d want 2", len(got))
	}
	t0, ok := got[0].(*Text)
	if !ok {
		t.Fatalf("got[0] = %T want *Text", got[0])
	}
	if t0.Text != "hello " {
		t.Errorf("got[0].Text = %q", t0.Text)
	}
	t1, ok := got[1].(*Text)
	if !ok {
		t.Fatalf("got[1] = %T want *Text", got[1])
	}
	if !t1.Bold {
		t.Errorf("got[1].Bold should be true")
	}
}

func TestDecodeInlineLink(t *testing.T) {
	raw := []byte(`[
		{"type":"text","text":"see "},
		{"type":"link","url":"https://example.com","children":[
			{"type":"text","text":"docs"}
		]},
		{"type":"text","text":" for details"}
	]`)
	got, err := decodeInline(raw)
	if err != nil {
		t.Fatalf("decodeInline: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d want 3", len(got))
	}
	link, ok := got[1].(*InlineLink)
	if !ok {
		t.Fatalf("got[1] = %T want *InlineLink", got[1])
	}
	if link.URL != "https://example.com" {
		t.Errorf("URL = %q", link.URL)
	}
	if len(link.Children) != 1 {
		t.Fatalf("link.Children len = %d", len(link.Children))
	}
	if link.Children[0].Text != "docs" {
		t.Errorf("link child = %q", link.Children[0].Text)
	}
}

func TestDecodeInlineErrorOnNonArray(t *testing.T) {
	_, err := decodeInline([]byte(`{"not":"array"}`))
	if err == nil {
		t.Fatal("expected error for non-array")
	}
}

func TestDecodeInlineSkipsUnknownTypes(t *testing.T) {
	// Unknown inline types are silently dropped (no logic in the SDK to
	// represent them); this protects against future Strapi additions.
	raw := []byte(`[
		{"type":"text","text":"keep"},
		{"type":"future-thing","payload":"drop me"},
		{"type":"text","text":" me"}
	]`)
	got, err := decodeInline(raw)
	if err != nil {
		t.Fatalf("decodeInline: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d want 2 (unknown dropped)", len(got))
	}
}

func TestInlineLinkUnmarshalDirect(t *testing.T) {
	// Directly unmarshaling into an *InlineLink should work via the standard
	// json package — this is what decodeInline does internally.
	raw := []byte(`{"type":"link","url":"https://x","children":[{"type":"text","text":"go"}]}`)
	var l InlineLink
	if err := json.Unmarshal(raw, &l); err != nil {
		t.Fatal(err)
	}
	if l.URL != "https://x" || len(l.Children) != 1 || l.Children[0].Text != "go" {
		t.Errorf("unexpected: %+v", l)
	}
}
