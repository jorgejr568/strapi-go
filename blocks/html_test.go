package blocks

import (
	"encoding/json"
	"strings"
	"testing"
)

func renderFixture(t *testing.T, src string) string {
	t.Helper()
	var bs Blocks
	if err := json.Unmarshal([]byte(src), &bs); err != nil {
		t.Fatal(err)
	}
	return RenderHTML(bs)
}

func TestRenderHTMLParagraph(t *testing.T) {
	got := renderFixture(t, `[{"type":"paragraph","children":[
		{"type":"text","text":"Hello "},
		{"type":"text","text":"world","bold":true}
	]}]`)
	want := "<p>Hello <strong>world</strong></p>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLHeading(t *testing.T) {
	got := renderFixture(t, `[{"type":"heading","level":2,"children":[{"type":"text","text":"Section"}]}]`)
	want := "<h2>Section</h2>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLUnorderedList(t *testing.T) {
	got := renderFixture(t, `[{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[{"type":"text","text":"a"}]},
		{"type":"list-item","children":[{"type":"text","text":"b"}]}
	]}]`)
	want := "<ul><li>a</li><li>b</li></ul>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLOrderedList(t *testing.T) {
	got := renderFixture(t, `[{"type":"list","format":"ordered","children":[
		{"type":"list-item","children":[{"type":"text","text":"a"}]}
	]}]`)
	want := "<ol><li>a</li></ol>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLLink(t *testing.T) {
	got := renderFixture(t, `[{"type":"link","url":"https://x","children":[{"type":"text","text":"go"}]}]`)
	want := `<a href="https://x">go</a>`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLCodeAndQuote(t *testing.T) {
	got := renderFixture(t, `[{"type":"code","children":[{"type":"text","text":"x := 1"}]}]`)
	if got != "<pre><code>x := 1</code></pre>" {
		t.Errorf("code got %q", got)
	}
	got = renderFixture(t, `[{"type":"quote","children":[{"type":"text","text":"hi"}]}]`)
	if got != "<blockquote>hi</blockquote>" {
		t.Errorf("quote got %q", got)
	}
}

func TestRenderHTMLImage(t *testing.T) {
	got := renderFixture(t, `[{"type":"image","image":{"url":"/u/x.jpg","alternativeText":"alt","width":800,"height":600},"children":[{"type":"text","text":""}]}]`)
	if !strings.Contains(got, `<img src="/u/x.jpg"`) {
		t.Errorf("img src missing in %q", got)
	}
	if !strings.Contains(got, `alt="alt"`) {
		t.Errorf("img alt missing in %q", got)
	}
}

func TestRenderHTMLEscapesContent(t *testing.T) {
	got := renderFixture(t, `[{"type":"paragraph","children":[{"type":"text","text":"<script>x</script>"}]}]`)
	want := "<p>&lt;script&gt;x&lt;/script&gt;</p>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLTextModifiers(t *testing.T) {
	got := renderFixture(t, `[{"type":"paragraph","children":[
		{"type":"text","text":"a","italic":true},
		{"type":"text","text":"b","underline":true},
		{"type":"text","text":"c","strikethrough":true},
		{"type":"text","text":"d","code":true}
	]}]`)
	want := "<p><em>a</em><u>b</u><s>c</s><code>d</code></p>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLParagraphWithInlineLink(t *testing.T) {
	// Inline link inside a paragraph exercises writeInlines's *InlineLink branch
	// and URL escaping.
	got := renderFixture(t, `[{"type":"paragraph","children":[
		{"type":"text","text":"see "},
		{"type":"link","url":"https://example.com/?q=1&x=2","children":[{"type":"text","text":"docs","bold":true}]},
		{"type":"text","text":" please"}
	]}]`)
	want := `<p>see <a href="https://example.com/?q=1&amp;x=2"><strong>docs</strong></a> please</p>`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLListItemWithInlineLink(t *testing.T) {
	// Exercises writeListItemChildren's *InlineLink branch, including URL
	// escaping and inner text rendering.
	got := renderFixture(t, `[{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[
			{"type":"text","text":"see "},
			{"type":"link","url":"https://example.com/?q=1&x=2","children":[{"type":"text","text":"docs","bold":true}]}
		]}
	]}]`)
	want := `<ul><li>see <a href="https://example.com/?q=1&amp;x=2"><strong>docs</strong></a></li></ul>`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLListItemWithNestedList(t *testing.T) {
	// Exercises writeListItemChildren's *List branch — nested lists recurse
	// through writeNode and emit a full <ul>/<ol> block inside the <li>.
	got := renderFixture(t, `[{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[
			{"type":"text","text":"outer "},
			{"type":"list","format":"ordered","children":[
				{"type":"list-item","children":[{"type":"text","text":"inner1"}]},
				{"type":"list-item","children":[{"type":"text","text":"inner2"}]}
			]}
		]},
		{"type":"list-item","children":[{"type":"text","text":"second"}]}
	]}]`)
	want := "<ul><li>outer <ol><li>inner1</li><li>inner2</li></ol></li><li>second</li></ul>"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRenderHTMLHeadingClampsLevel(t *testing.T) {
	// Heading levels outside 1-6 should clamp to h2 to keep output valid HTML.
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"zero", `[{"type":"heading","level":0,"children":[{"type":"text","text":"x"}]}]`, "<h2>x</h2>"},
		{"seven", `[{"type":"heading","level":7,"children":[{"type":"text","text":"x"}]}]`, "<h2>x</h2>"},
		{"negative", `[{"type":"heading","level":-1,"children":[{"type":"text","text":"x"}]}]`, "<h2>x</h2>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderFixture(t, tc.input)
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}
