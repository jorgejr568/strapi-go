package blocks

import (
	"encoding/json"
)

// InlineNode is any inline-level element that can appear inside an
// inline-bearing block's Children (Paragraph, Heading, Quote, Code, Link,
// Image, and ListItem). Implemented by *Text and *InlineLink. The
// inlineNodeType method is unexported, sealing the interface to types
// defined in this package.
type InlineNode interface {
	inlineNodeType() string
}

// inlineNodeType satisfies the InlineNode interface on Text. Text already
// implements Node (block-level) via the existing nodeType() method; the
// inline-level tag is additive.
func (Text) inlineNodeType() string { return "text" }

// InlineLink is a hyperlink inline node — the common form Strapi emits for
// links inside a paragraph or heading. Its Children is always []Text (links
// cannot nest links in Strapi's grammar).
type InlineLink struct {
	URL      string `json:"url"`
	Children []Text `json:"children"`
}

func (InlineLink) inlineNodeType() string { return "link" }

// decodeInline parses a JSON array of inline-child objects and dispatches
// each by its "type" discriminator into *Text or *InlineLink. Unknown
// inline types are silently dropped — they have no representation in the
// AST. The caller passes the raw bytes of an array; the function returns
// the typed slice or a decode error.
func decodeInline(data []byte) ([]InlineNode, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]InlineNode, 0, len(raw))
	for _, item := range raw {
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(item, &head); err != nil {
			return nil, err
		}
		switch head.Type {
		case "text":
			var n Text
			if err := json.Unmarshal(item, &n); err != nil {
				return nil, err
			}
			out = append(out, &n)
		case "link":
			var n InlineLink
			if err := json.Unmarshal(item, &n); err != nil {
				return nil, err
			}
			out = append(out, &n)
		default:
			// Silently drop unknown inline types.
		}
	}
	return out, nil
}
