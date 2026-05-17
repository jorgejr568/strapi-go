package blocks

import (
	"html"
	"strconv"
	"strings"
)

// RenderHTML emits a minimal, escaped HTML serialization of the blocks
// tree suitable for direct insertion into a page. The output is NOT a
// full document — no <html>/<body> wrappers.
func RenderHTML(bs Blocks) string {
	var sb strings.Builder
	for _, n := range bs {
		writeNode(&sb, n)
	}
	return sb.String()
}

func writeNode(sb *strings.Builder, n Node) {
	switch v := n.(type) {
	case *Paragraph:
		sb.WriteString("<p>")
		writeInlines(sb, v.Children)
		sb.WriteString("</p>")
	case *Heading:
		level := v.Level
		if level < 1 || level > 6 {
			level = 2
		}
		tag := "h" + strconv.Itoa(level)
		sb.WriteString("<")
		sb.WriteString(tag)
		sb.WriteString(">")
		writeInlines(sb, v.Children)
		sb.WriteString("</")
		sb.WriteString(tag)
		sb.WriteString(">")
	case *List:
		tag := "ul"
		if v.Format == "ordered" {
			tag = "ol"
		}
		sb.WriteString("<")
		sb.WriteString(tag)
		sb.WriteString(">")
		for _, item := range v.Items {
			sb.WriteString("<li>")
			writeTexts(sb, item.Children)
			sb.WriteString("</li>")
		}
		sb.WriteString("</")
		sb.WriteString(tag)
		sb.WriteString(">")
	case *Quote:
		sb.WriteString("<blockquote>")
		writeInlines(sb, v.Children)
		sb.WriteString("</blockquote>")
	case *Code:
		sb.WriteString("<pre><code>")
		writeInlines(sb, v.Children)
		sb.WriteString("</code></pre>")
	case *Link:
		sb.WriteString(`<a href="`)
		sb.WriteString(html.EscapeString(v.URL))
		sb.WriteString(`">`)
		writeTexts(sb, v.Children)
		sb.WriteString("</a>")
	case *Image:
		sb.WriteString(`<img src="`)
		sb.WriteString(html.EscapeString(v.Image.URL))
		sb.WriteString(`"`)
		if v.Image.AlternativeText != "" {
			sb.WriteString(` alt="`)
			sb.WriteString(html.EscapeString(v.Image.AlternativeText))
			sb.WriteString(`"`)
		}
		if v.Image.Width > 0 {
			sb.WriteString(` width="`)
			sb.WriteString(strconv.Itoa(v.Image.Width))
			sb.WriteString(`"`)
		}
		if v.Image.Height > 0 {
			sb.WriteString(` height="`)
			sb.WriteString(strconv.Itoa(v.Image.Height))
			sb.WriteString(`"`)
		}
		sb.WriteString(" />")
	case *Unknown:
		// Skip unknown nodes silently.
	}
}

// writeInlines renders a slice of inline nodes by dispatching on type.
// Each *Text is rendered with its modifiers; each *InlineLink is rendered
// as <a href="ESCAPED_URL">...</a> with its inner texts.
func writeInlines(sb *strings.Builder, nodes []InlineNode) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *Text:
			writeText(sb, *v)
		case *InlineLink:
			sb.WriteString(`<a href="`)
			sb.WriteString(html.EscapeString(v.URL))
			sb.WriteString(`">`)
			for _, t := range v.Children {
				writeText(sb, t)
			}
			sb.WriteString("</a>")
		}
	}
}

func writeTexts(sb *strings.Builder, texts []Text) {
	for _, t := range texts {
		writeText(sb, t)
	}
}

func writeText(sb *strings.Builder, t Text) {
	open, close := textTags(t)
	sb.WriteString(open)
	sb.WriteString(html.EscapeString(t.Text))
	sb.WriteString(close)
}

func textTags(t Text) (open, closeT string) {
	var openTags []string
	var closeTags []string
	if t.Bold {
		openTags = append(openTags, "<strong>")
		closeTags = append([]string{"</strong>"}, closeTags...)
	}
	if t.Italic {
		openTags = append(openTags, "<em>")
		closeTags = append([]string{"</em>"}, closeTags...)
	}
	if t.Underline {
		openTags = append(openTags, "<u>")
		closeTags = append([]string{"</u>"}, closeTags...)
	}
	if t.Strikethrough {
		openTags = append(openTags, "<s>")
		closeTags = append([]string{"</s>"}, closeTags...)
	}
	if t.Code {
		openTags = append(openTags, "<code>")
		closeTags = append([]string{"</code>"}, closeTags...)
	}
	return strings.Join(openTags, ""), strings.Join(closeTags, "")
}
