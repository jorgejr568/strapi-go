package blocks

import (
	"encoding/json"
	"strings"
	"testing"
)

const sampleBlocks = `[
  { "type": "paragraph", "children": [
    { "type": "text", "text": "Hello " },
    { "type": "text", "text": "world", "bold": true }
  ]},
  { "type": "heading", "level": 2, "children": [
    { "type": "text", "text": "Section" }
  ]},
  { "type": "list", "format": "unordered", "children": [
    { "type": "list-item", "children": [{ "type":"text", "text":"one" }] },
    { "type": "list-item", "children": [{ "type":"text", "text":"two" }] }
  ]},
  { "type": "link", "url": "https://example.com", "children": [
    { "type": "text", "text": "link text" }
  ]},
  { "type": "code", "children": [{ "type":"text", "text":"go run ." }] },
  { "type": "quote", "children": [{ "type":"text", "text":"quoted" }] },
  { "type": "image", "image": { "url":"/uploads/x.jpg", "alternativeText":"alt", "width":800, "height":600 }, "children":[{"type":"text","text":""}] }
]`

func TestBlocksUnmarshal(t *testing.T) {
	var bs Blocks
	if err := json.Unmarshal([]byte(sampleBlocks), &bs); err != nil {
		t.Fatal(err)
	}
	if len(bs) != 7 {
		t.Fatalf("blocks len = %d want 7", len(bs))
	}

	p, ok := bs[0].(*Paragraph)
	if !ok {
		t.Fatalf("bs[0] type = %T want *Paragraph", bs[0])
	}
	if len(p.Children) != 2 {
		t.Errorf("paragraph children = %d", len(p.Children))
	}
	// Children are now InlineNode; assert via type assertion.
	t1, ok := p.Children[1].(*Text)
	if !ok {
		t.Fatalf("p.Children[1] = %T want *Text", p.Children[1])
	}
	if !t1.Bold {
		t.Errorf("second text should be bold")
	}

	h, ok := bs[1].(*Heading)
	if !ok {
		t.Fatalf("bs[1] type = %T want *Heading", bs[1])
	}
	if h.Level != 2 {
		t.Errorf("heading level = %d", h.Level)
	}

	l, ok := bs[2].(*List)
	if !ok {
		t.Fatalf("bs[2] type = %T want *List", bs[2])
	}
	if l.Format != "unordered" {
		t.Errorf("list format = %q", l.Format)
	}
	if len(l.Items) != 2 {
		t.Errorf("list items = %d", len(l.Items))
	}

	link, ok := bs[3].(*Link)
	if !ok {
		t.Fatalf("bs[3] type = %T want *Link", bs[3])
	}
	if link.URL != "https://example.com" {
		t.Errorf("link URL = %q", link.URL)
	}

	if _, ok := bs[4].(*Code); !ok {
		t.Errorf("bs[4] type = %T want *Code", bs[4])
	}
	if _, ok := bs[5].(*Quote); !ok {
		t.Errorf("bs[5] type = %T want *Quote", bs[5])
	}

	img, ok := bs[6].(*Image)
	if !ok {
		t.Fatalf("bs[6] type = %T want *Image", bs[6])
	}
	if img.Image.URL != "/uploads/x.jpg" {
		t.Errorf("image URL = %q", img.Image.URL)
	}
}

func TestBlocksUnknownTypeBecomesRaw(t *testing.T) {
	src := `[{"type":"custom-thing","payload":{"x":1}}]`
	var bs Blocks
	if err := json.Unmarshal([]byte(src), &bs); err != nil {
		t.Fatal(err)
	}
	if len(bs) != 1 {
		t.Fatalf("len = %d", len(bs))
	}
	raw, ok := bs[0].(*Unknown)
	if !ok {
		t.Fatalf("type = %T want *Unknown", bs[0])
	}
	if raw.Type != "custom-thing" {
		t.Errorf("Type = %q", raw.Type)
	}
}

func TestNodeInterfaceConformance(t *testing.T) {
	// All concrete node types implement Node via an unexported nodeType()
	// method. Sealed-interface pattern: external types can't satisfy Node.
	// This test exercises each nodeType() to confirm the implementation
	// signature is correct and the type is recognized at the interface.
	values := []struct {
		name string
		node Node
	}{
		{"Text", Text{}},
		{"Paragraph", Paragraph{}},
		{"Heading", Heading{}},
		{"List", List{}},
		{"Quote", Quote{}},
		{"Code", Code{}},
		{"Link", Link{}},
		{"Image", Image{}},
		{"Unknown", Unknown{}},
	}
	for _, v := range values {
		t.Run(v.name, func(t *testing.T) {
			// Calling Node.nodeType through the interface forces the method-
			// set check at runtime. The package-internal `nodeType` is reached
			// because the test is in the same package.
			if v.node == nil {
				t.Fatal("nil Node")
			}
			_ = v.node // value's existence satisfies the static check
			// Reach into the method directly to cover the line.
			switch n := v.node.(type) {
			case Text:
				if n.nodeType() != "text" {
					t.Errorf("Text.nodeType() = %q", n.nodeType())
				}
			case Paragraph:
				if n.nodeType() != "paragraph" {
					t.Errorf("Paragraph.nodeType() = %q", n.nodeType())
				}
			case Heading:
				if n.nodeType() != "heading" {
					t.Errorf("Heading.nodeType() = %q", n.nodeType())
				}
			case List:
				if n.nodeType() != "list" {
					t.Errorf("List.nodeType() = %q", n.nodeType())
				}
			case Quote:
				if n.nodeType() != "quote" {
					t.Errorf("Quote.nodeType() = %q", n.nodeType())
				}
			case Code:
				if n.nodeType() != "code" {
					t.Errorf("Code.nodeType() = %q", n.nodeType())
				}
			case Link:
				if n.nodeType() != "link" {
					t.Errorf("Link.nodeType() = %q", n.nodeType())
				}
			case Image:
				if n.nodeType() != "image" {
					t.Errorf("Image.nodeType() = %q", n.nodeType())
				}
			case Unknown:
				if n.nodeType() != "unknown" {
					t.Errorf("Unknown.nodeType() = %q", n.nodeType())
				}
			default:
				t.Errorf("unhandled type %T", v.node)
			}
		})
	}
}

func TestBlocksUnmarshalReturnsErrorOnNonArray(t *testing.T) {
	var bs Blocks
	err := bs.UnmarshalJSON([]byte(`{"not": "an array"}`))
	if err == nil {
		t.Fatal("expected error for non-array top level")
	}
}

func TestBlocksUnmarshalWrapsPerNodeError(t *testing.T) {
	// First node has a valid type but its body fails to decode (level expects int).
	var bs Blocks
	err := bs.UnmarshalJSON([]byte(`[{"type":"heading","level":"not a number"}]`))
	if err == nil {
		t.Fatal("expected error for invalid heading body")
	}
	if !strings.Contains(err.Error(), "blocks[0]") {
		t.Errorf("error should be indexed with blocks[0]: %v", err)
	}
}

func TestBlocksUnmarshalErrorOnInvalidNodeShape(t *testing.T) {
	// Each item must be an object so the {"type":...} probe succeeds.
	var bs Blocks
	err := bs.UnmarshalJSON([]byte(`["a string, not an object"]`))
	if err == nil {
		t.Fatal("expected error for non-object element")
	}
}

func TestListUnmarshalReturnsErrorOnInvalidShape(t *testing.T) {
	var l List
	err := l.UnmarshalJSON([]byte(`{"format": 123}`)) // format must be string
	if err == nil {
		t.Fatal("expected error for invalid list format type")
	}
}

func TestParagraphWithInlineLink(t *testing.T) {
	raw := []byte(`{"type":"paragraph","children":[
		{"type":"text","text":"see "},
		{"type":"link","url":"https://example.com","children":[{"type":"text","text":"docs"}]},
		{"type":"text","text":" please"}
	]}`)
	var p Paragraph
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Children) != 3 {
		t.Fatalf("len = %d want 3", len(p.Children))
	}
	link, ok := p.Children[1].(*InlineLink)
	if !ok {
		t.Fatalf("Children[1] = %T want *InlineLink", p.Children[1])
	}
	if link.URL != "https://example.com" {
		t.Errorf("URL = %q", link.URL)
	}
	if link.Children[0].Text != "docs" {
		t.Errorf("link inner text = %q", link.Children[0].Text)
	}
}

func TestHeadingWithInlineLink(t *testing.T) {
	raw := []byte(`{"type":"heading","level":2,"children":[
		{"type":"text","text":"about "},
		{"type":"link","url":"/team","children":[{"type":"text","text":"us"}]}
	]}`)
	var h Heading
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	if h.Level != 2 || len(h.Children) != 2 {
		t.Fatalf("unexpected: %+v", h)
	}
	if _, ok := h.Children[1].(*InlineLink); !ok {
		t.Errorf("Children[1] = %T want *InlineLink", h.Children[1])
	}
}

func TestInlineBearingBlocksHandleEmptyAndMissingChildren(t *testing.T) {
	// Missing "children" key -> nil slice, no error.
	var p Paragraph
	if err := json.Unmarshal([]byte(`{"type":"paragraph"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Children != nil {
		t.Errorf("paragraph children = %v want nil", p.Children)
	}

	var h Heading
	if err := json.Unmarshal([]byte(`{"type":"heading","level":3}`), &h); err != nil {
		t.Fatal(err)
	}
	if h.Level != 3 || h.Children != nil {
		t.Errorf("heading = %+v", h)
	}

	var q Quote
	if err := json.Unmarshal([]byte(`{"type":"quote"}`), &q); err != nil {
		t.Fatal(err)
	}
	if q.Children != nil {
		t.Errorf("quote children = %v want nil", q.Children)
	}

	var c Code
	if err := json.Unmarshal([]byte(`{"type":"code"}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Children != nil {
		t.Errorf("code children = %v want nil", c.Children)
	}
}

func TestInlineBearingBlocksReturnErrorOnInvalidShape(t *testing.T) {
	// Top-level body must be a JSON object so the inner Unmarshal fails.
	var p Paragraph
	if err := p.UnmarshalJSON([]byte(`"not an object"`)); err == nil {
		t.Error("paragraph: expected error on non-object body")
	}
	var h Heading
	if err := h.UnmarshalJSON([]byte(`"not an object"`)); err == nil {
		t.Error("heading: expected error on non-object body")
	}
	var q Quote
	if err := q.UnmarshalJSON([]byte(`"not an object"`)); err == nil {
		t.Error("quote: expected error on non-object body")
	}
	var c Code
	if err := c.UnmarshalJSON([]byte(`"not an object"`)); err == nil {
		t.Error("code: expected error on non-object body")
	}
}

func TestInlineBearingBlocksPropagateDecodeInlineError(t *testing.T) {
	// children is present but not a JSON array -> decodeInline returns an error.
	bad := []byte(`{"type":"paragraph","children":{"oops":true}}`)
	var p Paragraph
	if err := p.UnmarshalJSON(bad); err == nil {
		t.Error("paragraph: expected error when children is not an array")
	}
	badH := []byte(`{"type":"heading","level":1,"children":{"oops":true}}`)
	var h Heading
	if err := h.UnmarshalJSON(badH); err == nil {
		t.Error("heading: expected error when children is not an array")
	}
	badQ := []byte(`{"type":"quote","children":{"oops":true}}`)
	var q Quote
	if err := q.UnmarshalJSON(badQ); err == nil {
		t.Error("quote: expected error when children is not an array")
	}
	badC := []byte(`{"type":"code","children":{"oops":true}}`)
	var c Code
	if err := c.UnmarshalJSON(badC); err == nil {
		t.Error("code: expected error when children is not an array")
	}
}

func TestQuoteAndCodeUseInlineChildren(t *testing.T) {
	q := []byte(`{"type":"quote","children":[{"type":"text","text":"q"}]}`)
	var quote Quote
	if err := json.Unmarshal(q, &quote); err != nil {
		t.Fatal(err)
	}
	if len(quote.Children) != 1 {
		t.Errorf("quote children = %d", len(quote.Children))
	}

	c := []byte(`{"type":"code","children":[{"type":"text","text":"x := 1"}]}`)
	var code Code
	if err := json.Unmarshal(c, &code); err != nil {
		t.Fatal(err)
	}
	if len(code.Children) != 1 {
		t.Errorf("code children = %d", len(code.Children))
	}
}
