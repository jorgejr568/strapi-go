package blocks

import (
	"encoding/json"
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
	if !p.Children[1].Bold {
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
