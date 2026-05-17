# Blocks Inline AST Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the MVP `Children []Text` limitation across the `blocks` AST with a proper sealed `InlineNode` interface, enabling correct decoding of inline links inside paragraphs/headings/quote/code/etc. and nested lists inside list-items. Update `RenderHTML` to emit inline `<a>` and nested `<ul>`/`<ol>`. Update the integration scenario and README. This is a pre-v0.1.0 breaking API change.

**Architecture:**
- New `InlineNode` interface in `blocks/` — sealed by an unexported `inlineNodeType()` method. Implemented by `Text` (existing, gains the new interface tag) and `InlineLink` (new). The existing `Text.nodeType() string` block-level interface method stays for backward compatibility with the block dispatcher, but `Text` is canonically used as an inline node going forward.
- All five inline-bearing block types (`Paragraph`, `Heading`, `Quote`, `Code`, `Link`) get their `Children` field switched from `[]Text` to `[]InlineNode`. Each gets a custom `UnmarshalJSON` that calls a new shared `decodeInline(data) ([]InlineNode, error)` helper to dispatch per-child by the `type` discriminator.
- `Image.Children` also switches to `[]InlineNode` for consistency; in practice this is almost always an empty/no-op array but the type unification is cheaper than carving out an exception.
- `ListItem` introduces a new `ListItemChild` interface that union-types `*Text`, `*InlineLink`, and `*List`. This matches Strapi's emit pattern: list items can mix inline text with nested block-level lists.
- `RenderHTML` learns three new emissions: `<a href="...">` for `*InlineLink` (inside any inline context), nested `<ul>`/`<ol>` inside `<li>` for `*List` in list-item children, and a small helper to render `[]InlineNode` consistently across all inline-bearing blocks.
- All existing tests that directly index into `Children` (e.g. `p.Children[1].Bold`) are migrated via type assertion (`p.Children[1].(*Text).Bold`).

**Tech Stack:** Go 1.22+, standard library only, no new test deps. Same patterns as existing blocks tests.

**Scope:**
- IN: inline `link` nodes inside any inline-bearing block; nested `list` nodes inside `list-item.children`; renderer support for both.
- IN: a clean breaking API change — `Children []Text` → `Children []InlineNode` everywhere, before v0.1.0 is tagged.
- OUT: arbitrary block-in-inline nesting (e.g. blockquote-inside-paragraph — Strapi doesn't emit this).
- OUT: inline-level images (Strapi emits images at block level only).
- OUT: a backward-compatible `[]Text` accessor — callers migrate to type assertions.

**Coverage target:** `blocks` package stays at 100% after the changes. Total weighted SDK coverage stays ≥92%.

---

## File Structure

```
blocks/
├── inline.go            # NEW — InlineNode interface + InlineLink type + decodeInline helper
├── blocks.go            # MODIFY — Children type changes + per-type custom UnmarshalJSON
├── html.go              # MODIFY — handle InlineLink + nested List inside ListItem
├── inline_test.go       # NEW — unit tests for decodeInline (discriminated decode)
├── blocks_test.go       # MODIFY — migrate existing tests to []InlineNode shape + new inline-link cases
└── html_test.go         # MODIFY — assert HTML for inline link + nested list
```

Plus:
- `lifecycle_test.go` — extend `TestBlocksRoundtrip` with inline-link and nested-list cases (one new scenario or expand the existing one).
- `README.md` — remove the "MVP limitation: Children []Text" note and update the rich-text section.

Responsibility per file:
- `inline.go`: the new inline-AST surface and its shared decoder helper.
- `blocks.go`: per-block-type `UnmarshalJSON` overrides that delegate inline-children decoding to `decodeInline`.
- `html.go`: updated renderer covering inline `<a>` and nested-list cases.

---

## Task 1: Add `InlineNode` interface, `InlineLink` type, and `decodeInline` helper

**Files:**
- Create: `blocks/inline.go`
- Create: `blocks/inline_test.go`

This task introduces the new types and the discriminated decoder. No existing code is touched yet — the existing `Children []Text` blocks keep working until Task 2.

The shared decoder is a free function `decodeInline(data []byte) ([]InlineNode, error)` that takes a raw JSON array of inline-child objects and returns the typed slice. It supports two `type` discriminators: `"text"` → `*Text`, `"link"` → `*InlineLink`. Unknown types are silently skipped (matches the block-level Unknown design but no need to expose them — inline nodes are simpler).

- [ ] **Step 1: Write the failing tests**

Create `blocks/inline_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `go test ./blocks/... -run "TestInlineLink|TestDecodeInline" -v`
Expected: compile error — `InlineNode`, `InlineLink`, `decodeInline` all undefined.

- [ ] **Step 3: Implement `blocks/inline.go`**

Create `blocks/inline.go`:

```go
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
			// Silently drop unknown inline types. They have no place in
			// the AST and we don't want to corrupt rendering with stubs.
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `go test ./blocks/... -run "TestInlineLink|TestDecodeInline" -v`
Expected: all 6 subtests PASS.

- [ ] **Step 5: Verify the full suite still green**

Run: `go test ./...`
Expected: all packages PASS. The existing blocks/blocks.go is unchanged so its tests still work.

- [ ] **Step 6: Commit**

```bash
git add blocks/inline.go blocks/inline_test.go
git commit -m "feat(blocks): add InlineNode interface and InlineLink type"
```

---

## Task 2: Switch `Paragraph`, `Heading`, `Quote`, `Code` to `[]InlineNode` children

**Files:**
- Modify: `blocks/blocks.go`
- Modify: `blocks/blocks_test.go`

These four block types share an identical shape — a single `Children` field of inline content. We change all four together because the JSON dispatcher in `decodeNode` already handles them uniformly, and a half-migration would leave the test suite inconsistent.

Each type gets a custom `UnmarshalJSON` that calls the auxiliary struct's `Children json.RawMessage` field and then `decodeInline` to produce `[]InlineNode`. The existing `TestBlocksUnmarshal` test that constructs `p.Children[1].Bold` is updated to use a type assertion.

- [ ] **Step 1: Update the failing test expectation in `blocks_test.go`**

In `blocks/blocks_test.go`, locate `TestBlocksUnmarshal`. Replace the paragraph-children assertion block (the one that reads `if !p.Children[1].Bold`) with:

```go
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
```

Heading, Quote, and Code assertions in `TestBlocksUnmarshal` don't index into Children's modifier fields, so they don't need updates beyond confirming the test still compiles.

Also, the existing assertion `bs[3].(*Link)` is for block-level Link — that gets handled in Task 3. Leave it alone for now (Task 3 will adjust).

- [ ] **Step 2: Add new tests that exercise inline links inside paragraph/heading/quote/code**

Append to `blocks/blocks_test.go`:

```go
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

func TestQuoteAndCodeUseInlineChildren(t *testing.T) {
	// Quote and Code now use []InlineNode too — sanity check they decode.
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
```

- [ ] **Step 3: Run tests and verify they fail**

Run: `go test ./blocks/... -v`
Expected: many compile errors. `Children` field on Paragraph/Heading/Quote/Code is still `[]Text`, so the new tests using `p.Children[1].(*InlineLink)` won't compile, and the modified TestBlocksUnmarshal will fail similarly.

- [ ] **Step 4: Update `blocks/blocks.go` — switch four types to `[]InlineNode`**

In `blocks/blocks.go`:

Change the field type on Paragraph, Heading, Quote, Code from `Children []Text` to `Children []InlineNode`. Mark the JSON tag with `"-"` (we'll decode via custom UnmarshalJSON):

```go
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
```

The existing `decodeNode` cases for `"paragraph"`, `"heading"`, `"quote"`, `"code"` already do `json.Unmarshal(data, &n)` — they automatically pick up the new custom UnmarshalJSON. No change needed in `decodeNode`.

- [ ] **Step 5: Run the test and verify it passes**

Run: `go test ./blocks/... -v`
Expected: PASS for all blocks tests including the new TestParagraphWithInlineLink / TestHeadingWithInlineLink / TestQuoteAndCodeUseInlineChildren and the modified TestBlocksUnmarshal.

The HTML renderer tests will likely fail — that's expected and gets fixed in Task 5. Note the failures and move on.

- [ ] **Step 6: Note: `html_test.go` likely fails — that's expected**

Run: `go test ./blocks/... -run TestRenderHTML`
Expected: failures, because the renderer's `writeTexts(sb, v.Children)` is now passing `[]InlineNode` to a `func(*strings.Builder, []Text)` — type mismatch.

Don't fix this yet. Task 5 owns the renderer changes.

- [ ] **Step 7: Run only the non-renderer blocks tests to verify Task 2's scope**

Run: `go test ./blocks/... -run "^(TestBlocksUnmarshal|TestBlocksUnknownTypeBecomesRaw|TestParagraphWithInlineLink|TestHeadingWithInlineLink|TestQuoteAndCodeUseInlineChildren|TestNodeInterfaceConformance|TestInlineLink|TestDecodeInline|TestBlocksUnmarshalReturnsError|TestBlocksUnmarshalWraps|TestBlocksUnmarshalErrorOnInvalid|TestListUnmarshalReturnsError)$" -v`
Expected: all PASS.

The package as a whole won't build (renderer fails) — that's OK temporarily until Task 5. Task 3 and 4 will compound on this state.

Actually wait — if the package doesn't compile, `go test ./blocks/...` will fail at the build step, not just at specific tests. Re-check: the renderer's `writeTexts` is called with `v.Children` where v is a `*Paragraph`. The new Paragraph.Children is `[]InlineNode`. The function signature `func(sb *strings.Builder, texts []Text)` no longer matches.

This is a build error. The package won't compile. We need to either:
A) Update the renderer in this task (combine with Task 5)
B) Add a temporary shim in this task to keep it compiling

Choose A — extending the scope of Task 2 to include a MINIMAL renderer update (just to keep it compiling). Task 5 then does the full renderer enhancement (inline links, nested lists).

Revise Step 4 to also include this minimal renderer patch in `blocks/html.go`:

In `blocks/html.go`, replace the `writeTexts` signature and body to accept `[]InlineNode`, and update all `writeTexts(sb, v.Children)` call sites accordingly. The `*Text` case keeps its existing behavior; the `*InlineLink` case can write a stub `<a>` tag (Task 5 polishes this).

Concrete patch (add to Task 2 Step 4 — modify `blocks/html.go`):

```go
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
```

And replace every `writeTexts(sb, v.Children)` call in `writeNode` with `writeInlines(sb, v.Children)`. Keep the old `writeTexts` function temporarily; it'll be removed in Task 5 once nothing references it.

Actually — wait. ListItem.Children is also `[]Text` currently and used in the renderer's List case (`writeTexts(sb, item.Children)`). Task 4 owns ListItem. So in Task 2, ListItem stays `[]Text` for now and the renderer still calls `writeTexts(sb, item.Children)` there. Keep `writeTexts` around until Tasks 4 and 5.

Similarly Image.Children, block-level Link.Children — they'll be touched in Task 3. Keep them on `[]Text` for now.

So Task 2's renderer patch is surgical: change only the 4 call sites for Paragraph, Heading, Quote, Code from `writeTexts` to `writeInlines`. The other 3 call sites (List items, Image, block-level Link) stay on `writeTexts([]Text)`.

- [ ] **Step 8: Run the full blocks test suite**

Run: `go test ./blocks/... -v`
Expected: all PASS. Renderer compiles. Inline-link tests pass. Existing TestRenderHTMLParagraph etc. still pass.

- [ ] **Step 9: Verify the full repo still green**

Run: `go test ./...`
Expected: all packages green.

- [ ] **Step 10: Commit**

```bash
git add blocks/blocks.go blocks/html.go blocks/blocks_test.go
git commit -m "feat(blocks): switch Paragraph/Heading/Quote/Code to InlineNode children"
```

---

## Task 3: Switch block-level `Link` and `Image` to `[]InlineNode` children

**Files:**
- Modify: `blocks/blocks.go`
- Modify: `blocks/html.go`
- Modify: `blocks/blocks_test.go`

Block-level `Link` (rarely emitted but valid per the SDK's contract) and `Image` (with its embedded image data) are the last two simple block types. Same migration pattern as Task 2.

- [ ] **Step 1: Update the existing test assertion for block-level Link**

In `TestBlocksUnmarshal` (`blocks_test.go`), the existing assertion is:

```go
	link, ok := bs[3].(*Link)
	if !ok {
		t.Fatalf("bs[3] type = %T want *Link", bs[3])
	}
	if link.URL != "https://example.com" {
		t.Errorf("link URL = %q", link.URL)
	}
```

The `URL` field doesn't change. Children isn't accessed by index here so no other adjustments are needed.

Add a new test specifically for block-level Link with inline children:

```go
func TestBlockLevelLinkChildrenAreInline(t *testing.T) {
	raw := []byte(`{"type":"link","url":"https://x","children":[
		{"type":"text","text":"go ","bold":true},
		{"type":"text","text":"home"}
	]}`)
	var l Link
	if err := json.Unmarshal(raw, &l); err != nil {
		t.Fatal(err)
	}
	if l.URL != "https://x" || len(l.Children) != 2 {
		t.Fatalf("unexpected: %+v", l)
	}
	t0, ok := l.Children[0].(*Text)
	if !ok || !t0.Bold {
		t.Errorf("Children[0] should be bold *Text, got %T", l.Children[0])
	}
}
```

And one for Image children (typically empty array but the field exists):

```go
func TestImageChildrenIsInlineNodeSlice(t *testing.T) {
	raw := []byte(`{"type":"image","image":{"url":"/u/x.jpg","alternativeText":"alt"},"children":[
		{"type":"text","text":""}
	]}`)
	var img Image
	if err := json.Unmarshal(raw, &img); err != nil {
		t.Fatal(err)
	}
	if img.Image.URL != "/u/x.jpg" {
		t.Errorf("Image.URL = %q", img.Image.URL)
	}
	if len(img.Children) != 1 {
		t.Errorf("len = %d", len(img.Children))
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run: `go test ./blocks/... -run "TestBlockLevelLinkChildren|TestImageChildren" -v`
Expected: compile errors — `l.Children[0].(*Text)` fails because Link.Children is still `[]Text`.

- [ ] **Step 3: Update `blocks/blocks.go` — switch Link and Image to `[]InlineNode`**

```go
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
```

- [ ] **Step 4: Update `blocks/html.go` — Link and Image renderer call sites**

In `writeNode`, find the `*Link` and `*Image` cases. Replace `writeTexts(sb, v.Children)` with `writeInlines(sb, v.Children)` for both.

- [ ] **Step 5: Run tests and verify they pass**

Run: `go test ./blocks/... -v`
Expected: all PASS. Inline-link, image, and block-level link tests all green.

- [ ] **Step 6: Verify the full repo still green**

Run: `go test ./...`
Expected: all packages green.

- [ ] **Step 7: Commit**

```bash
git add blocks/blocks.go blocks/html.go blocks/blocks_test.go
git commit -m "feat(blocks): switch Link and Image to InlineNode children"
```

---

## Task 4: `ListItem` — introduce `ListItemChild` interface, support nested lists

**Files:**
- Modify: `blocks/blocks.go`
- Modify: `blocks/html.go`
- Modify: `blocks/blocks_test.go`

List items have a unique constraint: their children can mix inline content (text, link) with nested block-level lists. Strapi emits patterns like:

```json
{"type":"list-item","children":[
    {"type":"text","text":"outer item "},
    {"type":"list","format":"unordered","children":[
        {"type":"list-item","children":[{"type":"text","text":"nested"}]}
    ]}
]}
```

To model this, we introduce a `ListItemChild` interface union-typing `*Text`, `*InlineLink`, and `*List`. A custom `UnmarshalJSON` on `ListItem` dispatches by the `type` discriminator.

- [ ] **Step 1: Update existing `TestBlocksUnmarshal` for nested-list-friendly assertions**

In `TestBlocksUnmarshal`, the existing list-items section reads:

```go
	if len(l.Items) != 2 {
		t.Errorf("list items = %d", len(l.Items))
	}
```

This works as-is (Items count unchanged). But the underlying ListItem.Children is changing from `[]Text` to `[]ListItemChild`. If the test doesn't index into `Items[0].Children` directly, no change needed. Verify by reading the file.

Add a new test for nested-list inside a list-item:

```go
func TestListItemSupportsNestedList(t *testing.T) {
	raw := []byte(`{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[
			{"type":"text","text":"outer "},
			{"type":"list","format":"unordered","children":[
				{"type":"list-item","children":[{"type":"text","text":"inner1"}]},
				{"type":"list-item","children":[{"type":"text","text":"inner2"}]}
			]}
		]},
		{"type":"list-item","children":[{"type":"text","text":"second top"}]}
	]}`)
	var outer List
	if err := json.Unmarshal(raw, &outer); err != nil {
		t.Fatal(err)
	}
	if len(outer.Items) != 2 {
		t.Fatalf("outer items = %d", len(outer.Items))
	}
	if len(outer.Items[0].Children) != 2 {
		t.Fatalf("first item children = %d want 2 (text + nested list)", len(outer.Items[0].Children))
	}
	// First child of the first item is a *Text.
	if _, ok := outer.Items[0].Children[0].(*Text); !ok {
		t.Errorf("Items[0].Children[0] = %T want *Text", outer.Items[0].Children[0])
	}
	// Second child is the nested *List.
	nested, ok := outer.Items[0].Children[1].(*List)
	if !ok {
		t.Fatalf("Items[0].Children[1] = %T want *List", outer.Items[0].Children[1])
	}
	if len(nested.Items) != 2 {
		t.Errorf("nested items = %d", len(nested.Items))
	}
	if _, ok := nested.Items[0].Children[0].(*Text); !ok {
		t.Errorf("nested Items[0].Children[0] should be *Text")
	}
}

func TestListItemSupportsInlineLinkInChildren(t *testing.T) {
	raw := []byte(`{"type":"list-item","children":[
		{"type":"text","text":"see "},
		{"type":"link","url":"https://x","children":[{"type":"text","text":"link"}]}
	]}`)
	var item ListItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatal(err)
	}
	if len(item.Children) != 2 {
		t.Fatalf("len = %d", len(item.Children))
	}
	if _, ok := item.Children[1].(*InlineLink); !ok {
		t.Errorf("Children[1] = %T want *InlineLink", item.Children[1])
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run: `go test ./blocks/... -run "TestListItemSupports" -v`
Expected: compile error — `outer.Items[0].Children[1].(*List)` fails (Children is still `[]Text`).

- [ ] **Step 3: Update `blocks/blocks.go` — ListItem with ListItemChild interface**

Add the new interface and update `ListItem`:

```go
// ListItemChild is anything that can appear inside a list-item's children:
// inline content (*Text, *InlineLink) and nested block-level lists (*List).
// The interface is sealed to types in this package.
type ListItemChild interface {
	isListItemChild()
}

func (Text) isListItemChild()       {}
func (InlineLink) isListItemChild() {}
func (List) isListItemChild()       {}

// ListItem is an element inside a List. Its Children can mix inline runs,
// inline links, and nested lists (Strapi emits all three patterns).
type ListItem struct {
	Children []ListItemChild `json:"-"`
}

// UnmarshalJSON decodes the "children" array by dispatching each item by
// its "type" discriminator into the appropriate concrete type.
func (li *ListItem) UnmarshalJSON(data []byte) error {
	var aux struct {
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	out := make([]ListItemChild, 0, len(aux.Children))
	for _, raw := range aux.Children {
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			return err
		}
		switch head.Type {
		case "text":
			var n Text
			if err := json.Unmarshal(raw, &n); err != nil {
				return err
			}
			out = append(out, &n)
		case "link":
			var n InlineLink
			if err := json.Unmarshal(raw, &n); err != nil {
				return err
			}
			out = append(out, &n)
		case "list":
			var n List
			if err := json.Unmarshal(raw, &n); err != nil {
				return err
			}
			out = append(out, &n)
		default:
			// Drop unknowns silently.
		}
	}
	li.Children = out
	return nil
}
```

- [ ] **Step 4: Update `blocks/html.go` — render ListItem children**

In `writeNode`'s `*List` case, the inner loop currently does:

```go
		for _, item := range v.Items {
			sb.WriteString("<li>")
			writeTexts(sb, item.Children)
			sb.WriteString("</li>")
		}
```

Replace with a new helper that dispatches on ListItemChild:

```go
		for _, item := range v.Items {
			sb.WriteString("<li>")
			writeListItemChildren(sb, item.Children)
			sb.WriteString("</li>")
		}
```

Add the helper to `blocks/html.go`:

```go
// writeListItemChildren renders the children of a list-item, dispatching on
// the union type ListItemChild: *Text and *InlineLink render inline,
// *List renders as a nested <ul> or <ol>.
func writeListItemChildren(sb *strings.Builder, children []ListItemChild) {
	for _, c := range children {
		switch v := c.(type) {
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
		case *List:
			writeNode(sb, v) // recurse — emits a full <ul>/<ol> block
		}
	}
}
```

The recursion through `writeNode` is safe because `writeNode`'s `*List` case handles all the rendering correctly (including any further nested lists).

- [ ] **Step 5: Run tests and verify they pass**

Run: `go test ./blocks/... -v`
Expected: all PASS, including the new TestListItemSupportsNestedList and TestListItemSupportsInlineLinkInChildren.

- [ ] **Step 6: Verify the full repo still green**

Run: `go test ./...`
Expected: all packages green.

- [ ] **Step 7: Commit**

```bash
git add blocks/blocks.go blocks/html.go blocks/blocks_test.go
git commit -m "feat(blocks): support nested lists and inline links in list-items"
```

---

## Task 5: HTML renderer — inline link rendering + cleanup unused helpers

**Files:**
- Modify: `blocks/html.go`
- Modify: `blocks/html_test.go`

By the end of Task 4, every block type has been migrated. The renderer compiles. But the old `writeTexts(sb *strings.Builder, texts []Text)` function is now used in only one place: inside the new `writeInlines` and `writeListItemChildren` helpers, and within `writeNode`'s `*InlineLink` case (actually they're calling `writeText` per-text, not `writeTexts`). Audit and remove dead code, then add HTML renderer tests for inline link + nested list output.

- [ ] **Step 1: Audit `blocks/html.go` for dead helpers**

Read `blocks/html.go`. Look for:
- `writeTexts(sb *strings.Builder, texts []Text)` — was used in every block-level child loop. After Tasks 2-4, those call sites all use `writeInlines` or `writeListItemChildren`. If no caller remains, remove `writeTexts` entirely.

Verify by grep: `grep -n writeTexts blocks/html.go` — should return zero hits after Task 4.

If `writeTexts` is unused, delete it. Otherwise leave it.

- [ ] **Step 2: Add HTML renderer tests for inline link inside paragraph**

Append to `blocks/html_test.go`:

```go
func TestRenderHTMLParagraphWithInlineLink(t *testing.T) {
	got := renderFixture(t, `[{"type":"paragraph","children":[
		{"type":"text","text":"see "},
		{"type":"link","url":"https://example.com","children":[{"type":"text","text":"docs"}]},
		{"type":"text","text":" please"}
	]}]`)
	want := `<p>see <a href="https://example.com">docs</a> please</p>`
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestRenderHTMLInlineLinkEscapesURL(t *testing.T) {
	got := renderFixture(t, `[{"type":"paragraph","children":[
		{"type":"link","url":"https://x?a=1&b=2","children":[{"type":"text","text":"go"}]}
	]}]`)
	if !strings.Contains(got, `href="https://x?a=1&amp;b=2"`) {
		t.Errorf("URL ampersand should be escaped, got %q", got)
	}
}

func TestRenderHTMLHeadingWithInlineLink(t *testing.T) {
	got := renderFixture(t, `[{"type":"heading","level":2,"children":[
		{"type":"text","text":"About "},
		{"type":"link","url":"/team","children":[{"type":"text","text":"us"}]}
	]}]`)
	want := `<h2>About <a href="/team">us</a></h2>`
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
```

- [ ] **Step 3: Add HTML renderer tests for nested list inside list-item**

Append:

```go
func TestRenderHTMLNestedList(t *testing.T) {
	got := renderFixture(t, `[{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[
			{"type":"text","text":"outer "},
			{"type":"list","format":"unordered","children":[
				{"type":"list-item","children":[{"type":"text","text":"inner1"}]},
				{"type":"list-item","children":[{"type":"text","text":"inner2"}]}
			]}
		]},
		{"type":"list-item","children":[{"type":"text","text":"second"}]}
	]}]`)
	want := `<ul><li>outer <ul><li>inner1</li><li>inner2</li></ul></li><li>second</li></ul>`
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestRenderHTMLListItemWithInlineLink(t *testing.T) {
	got := renderFixture(t, `[{"type":"list","format":"unordered","children":[
		{"type":"list-item","children":[
			{"type":"text","text":"check "},
			{"type":"link","url":"https://x","children":[{"type":"text","text":"this"}]}
		]}
	]}]`)
	want := `<ul><li>check <a href="https://x">this</a></li></ul>`
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
```

- [ ] **Step 4: Run renderer tests and verify they pass**

Run: `go test ./blocks/... -run TestRenderHTML -v`
Expected: all renderer tests PASS, including the 5 new ones.

- [ ] **Step 5: Verify the full repo still green**

Run: `go test ./...`
Expected: all packages green.

- [ ] **Step 6: Coverage check**

Run: `go test -cover ./blocks/...`
Expected: `blocks` coverage at 100% (was 100% before; the new code is fully tested by Tasks 1-5).

- [ ] **Step 7: Commit**

```bash
git add blocks/html.go blocks/html_test.go
git commit -m "feat(blocks): render inline links and nested lists; remove dead helpers"
```

---

## Task 6: Update integration scenario + README

**Files:**
- Modify: `lifecycle_test.go`
- Modify: `README.md`

Extend the existing `TestBlocksRoundtrip` integration scenario with the new inline-link and nested-list cases so the end-to-end roundtrip exercises them. Remove the MVP limitation note from the README's rich-text section.

- [ ] **Step 1: Extend `TestBlocksRoundtrip` in `lifecycle_test.go`**

Find the JSON body inside the `srv` handler (the multi-block document the test renders). Insert two new blocks into the `body` array — one paragraph containing an inline link, and one list with a nested list. Place them after the existing list blocks and before the link block. The body array should grow from 9 elements to 11.

Concretely, locate this section in `lifecycle_test.go` (inside `TestBlocksRoundtrip`):

```go
                    {"type":"list","format":"ordered","children":[
                        {"type":"list-item","children":[{"type":"text","text":"one"}]}
                    ]},
                    {"type":"quote","children":[{"type":"text","text":"quoted"}]},
```

Insert between them (after the ordered-list block, before the quote block):

```go
                    {"type":"paragraph","children":[
                        {"type":"text","text":"with "},
                        {"type":"link","url":"https://example.com/docs","children":[{"type":"text","text":"inline link"}]},
                        {"type":"text","text":" here"}
                    ]},
                    {"type":"list","format":"unordered","children":[
                        {"type":"list-item","children":[
                            {"type":"text","text":"outer "},
                            {"type":"list","format":"unordered","children":[
                                {"type":"list-item","children":[{"type":"text","text":"nested"}]}
                            ]}
                        ]}
                    ]},
```

Update the body-length assertion: change `if len(page.Attributes.Body) != 9` to `if len(page.Attributes.Body) != 11`.

Add the new expected substrings to `mustContain`:

```go
	mustContain := []string{
		"<h1>Title</h1>",
		"<p>plain ",
		"<strong>bold</strong>",
		"<em> italic</em>",
		"<strong><em> both</em></strong>",
		"<ul><li>first</li><li>second</li></ul>",
		"<ol><li>one</li></ol>",
		// New: inline link inside paragraph
		`<a href="https://example.com/docs">inline link</a>`,
		// New: nested list inside list-item
		`<ul><li>outer <ul><li>nested</li></ul></li></ul>`,
		"<blockquote>quoted</blockquote>",
		"<pre><code>go run .</code></pre>",
		`<a href="https://example.com">site</a>`,
		`<img src="/uploads/x.jpg"`,
		`alt="alt"`,
		`width="800"`,
		`height="600"`,
	}
```

- [ ] **Step 2: Run the lifecycle test**

Run: `go test -run TestBlocksRoundtrip -v`
Expected: PASS. All 16 expected substrings present.

- [ ] **Step 3: Update `README.md`**

In `README.md`, find the "Rich-text blocks" section. Remove the limitation paragraph that begins:

> **Limitation:** the MVP types `Paragraph.Children`, `Heading.Children`, …

Also remove the corresponding bullet in the "Not yet" section that mentions "Inline links and nested lists … decoded as plain text only."

Replace with a brief positive statement:

```markdown
## Rich-text blocks

`blocks.Blocks` decodes Strapi v4.13+ / v5 rich-text JSON into a typed
AST. Pass it to `blocks.RenderHTML` for a minimal escaped HTML
serialization, or walk it yourself for custom rendering.

The inline content of `Paragraph`, `Heading`, `Quote`, `Code`, `Link`,
`Image`, and `ListItem` children is decoded as `[]blocks.InlineNode`
(sealed by the package): the concrete types are `*blocks.Text` for plain
runs (with bold/italic/etc. modifiers) and `*blocks.InlineLink` for
inline hyperlinks. List-item children additionally accept `*blocks.List`
for nested lists. Use type assertions to access concrete fields:

`​`​`go
for _, child := range para.Children {
    switch n := child.(type) {
    case *blocks.Text:
        fmt.Println(n.Text, n.Bold)
    case *blocks.InlineLink:
        fmt.Println(n.URL, n.Children[0].Text)
    }
}
`​`​`
```

- [ ] **Step 4: Run the full suite**

Run: `go test ./... && go vet ./...`
Expected: all green, vet clean.

- [ ] **Step 5: Commit**

```bash
git add lifecycle_test.go README.md
git commit -m "test: extend blocks roundtrip with inline link and nested list; update README"
```

---

## Task 7: Final verification

**Files:**
- None (read-only)

Run the whole suite with race detector, confirm coverage holds, do a quick scan for any leftover references to the old `[]Text` Children type that may have been missed.

- [ ] **Step 1: Race-detector pass**

Run: `go test -race ./...`
Expected: all packages PASS clean under `-race`.

- [ ] **Step 2: Coverage profile**

Run: `go test -coverprofile=/tmp/cov-blocks-inline.out ./... && go test -cover ./...`
Expected:
- `blocks` ≥99% (preferably 100%).
- All other packages unchanged from prior baseline.
- Total weighted ≥92%.

- [ ] **Step 3: Scan for leftover `[]Text` references**

Run: `grep -rn "Children []Text" blocks/`
Expected: zero hits. The only remaining `[]Text` should be `InlineLink.Children` (links contain only text per Strapi grammar).

Run: `grep -rn "\[\]Text" blocks/`
Expected: hits only on `InlineLink.Children []Text` and on the `EmbeddedImage` struct (unrelated — it's a sibling of `Image.Children`).

If any unexpected leftover `[]Text` exists in inline-bearing blocks, fix it. Otherwise skip.

- [ ] **Step 4: Type-assertion sanity check**

Write a tiny smoke-test program at `/tmp/smoke.go` (not committed) that imports the SDK and walks a `Paragraph.Children` with a type switch — confirming the public API ergonomics are sane:

```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/jorgejr568/strapi-go/blocks"
)

func main() {
	raw := []byte(`{"type":"paragraph","children":[
		{"type":"text","text":"hi "},
		{"type":"link","url":"https://x","children":[{"type":"text","text":"link"}]}
	]}`)
	var p blocks.Paragraph
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(err)
	}
	for _, c := range p.Children {
		switch n := c.(type) {
		case *blocks.Text:
			fmt.Printf("text: %q (bold=%v)\n", n.Text, n.Bold)
		case *blocks.InlineLink:
			fmt.Printf("link: %s -> %d children\n", n.URL, len(n.Children))
		}
	}
}
```

Run: `cd /tmp && go run smoke.go` (after a temporary `go mod init smoke && go mod edit -replace github.com/jorgejr568/strapi-go=/Users/j/src/jorgejr568/strapi-go && go mod tidy`)

Expected output:
```
text: "hi " (bold=false)
link: https://x -> 1 children
```

Delete the smoke harness after verification.

- [ ] **Step 5: Final commit (only if any fixes were needed in Step 3)**

If Step 3 surfaced cleanup, commit:

```bash
git add blocks/
git commit -m "fix(blocks): clean up leftover []Text references"
```

Otherwise no commit — this is a verification-only task.

---

## Self-Review

**Spec coverage:**
- Inline links inside paragraph/heading/quote/code/block-Link/image — Tasks 2 and 3. ✅
- Nested lists inside list-items — Task 4. ✅
- HTML renderer emits `<a>` and nested `<ul>/<ol>` — Tasks 2, 4, and 5. ✅
- Breaking API change documented and consistent across all consumers — Task 6 (README) + the migrated existing tests in Tasks 2-4. ✅
- Coverage held at ≥99% on blocks package — Task 5 Step 6 + Task 7 Step 2. ✅
- Integration scenario exercises the new shapes end-to-end — Task 6. ✅

**Type consistency check:**
- `InlineNode` interface defined in Task 1; used as the Children type in Tasks 2 and 3. ✅
- `ListItemChild` interface defined in Task 4; used only by `ListItem`. ✅
- `*Text` and `*InlineLink` both satisfy both `InlineNode` (Tasks 1, 2, 3) and `ListItemChild` (Task 4). ✅
- `*List` satisfies `ListItemChild` (Task 4) and is reused as-is in `writeListItemChildren`. ✅
- `decodeInline` (Task 1) is called from Paragraph/Heading/Quote/Code (Task 2) and Link/Image (Task 3) custom UnmarshalJSON. ✅
- `writeInlines` (Task 2) and `writeListItemChildren` (Task 4) cover all inline rendering surfaces. ✅

**Placeholder scan:**
- No TBDs, no "implement later", no naked references. Every test, type, and helper is fully spelled out.
- Task 2 explicitly notes the transitional state where renderer compiles only after Step 7's minimal renderer patch — flagged with the rationale.
- Task 7's smoke-test isn't committed; the plan explicitly says delete it after verification.

**One trade-off worth flagging during execution:**
The plan keeps the legacy block-level `Link` type (Task 3) even though Strapi rarely emits links at the block level. An alternative was to remove block-level `Link` entirely and only support `InlineLink`. Keeping it preserves backward compatibility with any existing fixtures and is cheap; removing it would shrink the API surface but force consumers to relearn the structure. Chosen path is the conservative one.

Plan is ready for execution.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-17-blocks-inline-ast.md`. Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task with two-stage review, fast iteration.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
