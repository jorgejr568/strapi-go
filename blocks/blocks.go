// Package blocks models the Strapi v4.13+/v5 rich-text "blocks" JSON
// structure as a Go AST and exposes a renderer interface.
package blocks

import (
	"encoding/json"
	"fmt"
)

// Node is any block-level element in a Blocks tree.
type Node interface {
	nodeType() string
}

// Blocks is the top-level array of block-level nodes returned by Strapi's
// "blocks" rich-text field. Use json.Unmarshal to decode.
type Blocks []Node

// Text is an inline run with optional formatting modifiers.
type Text struct {
	Text          string `json:"text"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Code          bool   `json:"code,omitempty"`
}

func (Text) nodeType() string { return "text" }

// Paragraph is a paragraph of inline children.
type Paragraph struct {
	Children []Text `json:"children"`
}

func (Paragraph) nodeType() string { return "paragraph" }

// Heading is a heading element. Level is 1–6.
type Heading struct {
	Level    int    `json:"level"`
	Children []Text `json:"children"`
}

func (Heading) nodeType() string { return "heading" }

// ListItem is an element inside a List.
type ListItem struct {
	Children []Text `json:"children"`
}

// List is an ordered or unordered list.
type List struct {
	Format string     `json:"format"` // "ordered" | "unordered"
	Items  []ListItem `json:"-"`
}

func (List) nodeType() string { return "list" }

// UnmarshalJSON maps the raw "children" array onto Items.
func (l *List) UnmarshalJSON(data []byte) error {
	var aux struct {
		Format   string     `json:"format"`
		Children []ListItem `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	l.Format = aux.Format
	l.Items = aux.Children
	return nil
}

// Quote is a block quotation.
type Quote struct {
	Children []Text `json:"children"`
}

func (Quote) nodeType() string { return "quote" }

// Code is a code block.
type Code struct {
	Children []Text `json:"children"`
}

func (Code) nodeType() string { return "code" }

// Link is a block-level link. (Strapi emits links at the block level in
// addition to the text-modifier form.)
type Link struct {
	URL      string `json:"url"`
	Children []Text `json:"children"`
}

func (Link) nodeType() string { return "link" }

// EmbeddedImage is the inline image payload embedded in an Image block.
type EmbeddedImage struct {
	URL             string `json:"url"`
	AlternativeText string `json:"alternativeText,omitempty"`
	Caption         string `json:"caption,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
}

// Image is an inline image block.
type Image struct {
	Image    EmbeddedImage `json:"image"`
	Children []Text        `json:"children"`
}

func (Image) nodeType() string { return "image" }

// Unknown preserves any block type the SDK doesn't recognize so consumers
// can inspect or pass it through. Type carries the discriminator;
// Raw carries the original JSON.
type Unknown struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

func (Unknown) nodeType() string { return "unknown" }

// UnmarshalJSON dispatches each item in the array to the concrete node
// type based on the "type" discriminator.
func (b *Blocks) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(Blocks, 0, len(raw))
	for i, item := range raw {
		node, err := decodeNode(item)
		if err != nil {
			return fmt.Errorf("blocks[%d]: %w", i, err)
		}
		out = append(out, node)
	}
	*b = out
	return nil
}

func decodeNode(data json.RawMessage) (Node, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, err
	}
	switch head.Type {
	case "paragraph":
		var n Paragraph
		return &n, json.Unmarshal(data, &n)
	case "heading":
		var n Heading
		return &n, json.Unmarshal(data, &n)
	case "list":
		var n List
		return &n, json.Unmarshal(data, &n)
	case "quote":
		var n Quote
		return &n, json.Unmarshal(data, &n)
	case "code":
		var n Code
		return &n, json.Unmarshal(data, &n)
	case "link":
		var n Link
		return &n, json.Unmarshal(data, &n)
	case "image":
		var n Image
		return &n, json.Unmarshal(data, &n)
	default:
		return &Unknown{Type: head.Type, Raw: append([]byte(nil), data...)}, nil
	}
}
