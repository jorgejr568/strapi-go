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
	Children []InlineNode `json:"-"`
}

func (Paragraph) nodeType() string { return "paragraph" }

// UnmarshalJSON decodes the "children" array into []InlineNode via the
// shared inline dispatcher.
func (p *Paragraph) UnmarshalJSON(data []byte) error {
	var aux struct {
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Children) == 0 {
		p.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	p.Children = children
	return nil
}

// Heading is a heading element. Level is 1–6.
type Heading struct {
	Level    int          `json:"level"`
	Children []InlineNode `json:"-"`
}

func (Heading) nodeType() string { return "heading" }

func (h *Heading) UnmarshalJSON(data []byte) error {
	var aux struct {
		Level    int             `json:"level"`
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	h.Level = aux.Level
	if len(aux.Children) == 0 {
		h.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	h.Children = children
	return nil
}

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
	Children []InlineNode `json:"-"`
}

func (Quote) nodeType() string { return "quote" }

func (q *Quote) UnmarshalJSON(data []byte) error {
	var aux struct {
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Children) == 0 {
		q.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	q.Children = children
	return nil
}

// Code is a code block.
type Code struct {
	Children []InlineNode `json:"-"`
}

func (Code) nodeType() string { return "code" }

func (c *Code) UnmarshalJSON(data []byte) error {
	var aux struct {
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Children) == 0 {
		c.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	c.Children = children
	return nil
}

// Link is a block-level link. (Strapi emits links at the block level in
// addition to the text-modifier form.)
type Link struct {
	URL      string       `json:"url"`
	Children []InlineNode `json:"-"`
}

func (Link) nodeType() string { return "link" }

func (l *Link) UnmarshalJSON(data []byte) error {
	var aux struct {
		URL      string          `json:"url"`
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	l.URL = aux.URL
	if len(aux.Children) == 0 {
		l.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	l.Children = children
	return nil
}

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
	Children []InlineNode  `json:"-"`
}

func (Image) nodeType() string { return "image" }

func (i *Image) UnmarshalJSON(data []byte) error {
	var aux struct {
		Image    EmbeddedImage   `json:"image"`
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	i.Image = aux.Image
	if len(aux.Children) == 0 {
		i.Children = nil
		return nil
	}
	children, err := decodeInline(aux.Children)
	if err != nil {
		return err
	}
	i.Children = children
	return nil
}

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
