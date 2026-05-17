# Strapi-Go SDK — Pages MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a public, idiomatic Go SDK that lets other projects read and mutate Strapi v5 content (collection types like Pages and single types like Homepage) as typed Go documents, with a fluent query builder, parsed rich-text blocks, and resolved media URLs.

**Architecture:** Top-level `strapi` package exposes a `Client` (functional options for base URL, API token, HTTP client) plus two generic accessors: `Collection[T]` for plural collection-type endpoints (`Find`, `List`, `Create`, `Update`, `Delete`) and `SingleType[T]` for the single-record single-type endpoints (`Get`, `Update`, `Delete`). A `strapi/query` subpackage offers a typed builder that emits Strapi-compatible `qs`-style bracket query strings (populate, filters, sort, pagination, fields, locale, status). A `strapi/blocks` subpackage parses Strapi v4.13+/v5 rich-text block JSON into an AST with an HTML renderer. A `strapi/media` subpackage models uploads and resolves relative URLs against the client's base URL. Responses are decoded via a generic `Document[T]` envelope that uses a two-pass `UnmarshalJSON` to split Strapi v5's flat system fields (`id`, `documentId`, timestamps, `locale`) from user-defined content fields. A single internal `do` helper handles all requests — it takes a method, path, raw query string, optional JSON body, and optional decode target, so reads and writes share one execution path with one error-decode site. Everything is stdlib-only — no third-party runtime or test deps.

**Tech Stack:**
- Go 1.22+ (generics, modern stdlib)
- Standard library only (`net/http`, `encoding/json`, `net/url`, `testing`, `net/http/httptest`)
- Module path: `github.com/jorgejr568/strapi-go`
- Target: Strapi v5 REST API (`documentId` as canonical key, flat response shape)

**Out of scope for this MVP** (note explicitly so future plans can pick them up):
- File upload (`POST /api/upload`)
- v4 response-format compatibility (`Strapi-Response-Format: v4` header)
- Dynamic-zone typed registry (raw `json.RawMessage` is fine for now)
- Markdown renderer for blocks (HTML only for MVP)
- Users-permissions auto-login flow (API token only)

---

## File Structure

```
github.com/jorgejr568/strapi-go/
├── go.mod                         # module path + Go version
├── .gitignore
├── LICENSE                        # MIT
├── README.md                      # usage + examples
├── doc.go                         # package strapi godoc
├── client.go                      # Client struct + New + options
├── options.go                     # functional Option type
├── document.go                    # Document[T], List[T], single[T], Meta, Pagination
├── errors.go                      # Error struct + sentinel errors
├── do.go                          # internal request/response helper (body-aware)
├── collection.go                  # Collection[T] + Find/List/Create/Update/Delete
├── single_type.go                 # SingleType[T] + Get/Update/Delete
├── client_test.go
├── document_test.go
├── errors_test.go
├── do_test.go
├── collection_test.go
├── collection_mutations_test.go
├── single_type_test.go
├── testdata/
│   ├── page_single.json
│   ├── page_list.json
│   └── error_404.json
├── query/
│   ├── query.go                   # Query, Option, basic params
│   ├── filter.go                  # Eq/Ne/.../And/Or/Not
│   ├── populate.go                # PopulateAll, PopulateField, builder
│   ├── encode.go                  # internal qs encoder
│   └── *_test.go
├── blocks/
│   ├── blocks.go                  # Block AST types + UnmarshalJSON
│   ├── html.go                    # HTML renderer
│   └── *_test.go
├── media/
│   ├── media.go                   # File, Format, ResolveURL
│   └── media_test.go
└── examples/
    └── fetch_pages/
        └── main.go
```

Responsibility per file:
- `client.go` / `options.go`: HTTP client wiring, configuration
- `do.go`: single internal `do` function that executes a request, optionally marshals a JSON body, decodes 2xx into `dst` or non-2xx into `*Error`
- `document.go`: response envelope types and the two-pass `UnmarshalJSON` trick
- `collection.go`: generic plural-endpoint accessor (CRUD)
- `single_type.go`: generic single-type accessor (read + update + delete)
- `query/*`: query builder; `encode.go` is internal to the package
- `blocks/*`: rich-text AST and renderer (independent of HTTP)
- `media/*`: media types and URL helper (independent of HTTP)

---

## Task 1: Project setup (module path, Go version, license, gitignore)

**Files:**
- Modify: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `doc.go`
- Delete: `options.go` (old throwaway stub), `responses.go` (old throwaway stub)

The existing `options.go` and `responses.go` are pre-existing throwaway stubs from a never-started attempt. Delete them — they hardcode v4 shapes and have a duplicated-condition bug in `handleClientOptions`. Everything will be re-built clean.

- [ ] **Step 1: Delete the throwaway stubs**

```bash
rm options.go responses.go
```

- [ ] **Step 2: Rewrite `go.mod` with the public module path and Go 1.22**

Overwrite `go.mod` to:

```
module github.com/jorgejr568/strapi-go

go 1.22
```

- [ ] **Step 3: Create `.gitignore`**

Create `.gitignore`:

```
# Binaries
*.exe
*.test
*.out
/bin/
/dist/

# Go
vendor/
coverage.out
coverage.html

# Editor / OS
.DS_Store
.idea/
.vscode/
*.swp
```

- [ ] **Step 4: Create `LICENSE` (MIT)**

Create `LICENSE`:

```
MIT License

Copyright (c) 2026 Jorge Junior

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

- [ ] **Step 5: Create `doc.go` (package-level godoc)**

Create `doc.go`:

```go
// Package strapi is a Go SDK for the Strapi v5 REST API.
//
// It exposes a typed Client and two generic accessors — Collection[T] for
// plural collection-type endpoints (e.g. Pages, Articles) and SingleType[T]
// for single-record single-type endpoints (e.g. Homepage, Footer) — with a
// fluent query builder under github.com/jorgejr568/strapi-go/query, a
// rich-text block parser under .../blocks, and media helpers under .../media.
//
// MVP scope: read + write access to collection types (Find, List, Create,
// Update, Delete) and single types (Get, Update, Delete). File uploads are
// not yet supported.
package strapi
```

- [ ] **Step 6: Verify the module builds**

Run: `go build ./...`
Expected: exit 0, no output (no packages compiled because no .go files yet other than doc.go).

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: bootstrap public module, drop throwaway stubs"
```

---

## Task 2: Errors — `Error` type and sentinels

**Files:**
- Create: `errors.go`
- Create: `errors_test.go`

- [ ] **Step 1: Write the failing test**

Create `errors_test.go`:

```go
package strapi

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	e := &Error{Status: 404, Name: "NotFoundError", Message: "Not Found"}
	got := e.Error()
	want := "strapi: NotFoundError (404): Not Found"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestErrorIs(t *testing.T) {
	cases := []struct {
		name     string
		err      *Error
		sentinel error
		want     bool
	}{
		{"404 by status", &Error{Status: 404}, ErrNotFound, true},
		{"404 by name", &Error{Name: "NotFoundError"}, ErrNotFound, true},
		{"401 by status", &Error{Status: 401}, ErrUnauthorized, true},
		{"403 by status", &Error{Status: 403}, ErrForbidden, true},
		{"400 validation", &Error{Status: 400, Name: "ValidationError"}, ErrValidation, true},
		{"mismatch", &Error{Status: 500}, ErrNotFound, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.err, tc.sentinel); got != tc.want {
				t.Fatalf("errors.Is = %v want %v", got, tc.want)
			}
		})
	}
}

func TestErrorDetailsPreserved(t *testing.T) {
	raw := json.RawMessage(`{"errors":[{"path":["title"],"message":"required"}]}`)
	e := &Error{Status: 400, Name: "ValidationError", Message: "fail", Details: raw}
	if string(e.Details) != string(raw) {
		t.Fatalf("details not preserved")
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./... -run TestError -v`
Expected: compile error — `Error`, `ErrNotFound`, etc. are undefined.

- [ ] **Step 3: Implement `errors.go`**

Create `errors.go`:

```go
package strapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Error is the typed error returned for any non-2xx Strapi response. It
// carries the full payload from Strapi's error envelope so callers can
// inspect details. Use errors.Is with the package sentinels for common
// branching.
type Error struct {
	Status  int             `json:"status"`
	Name    string          `json:"name"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("strapi: %s (%d): %s", e.Name, e.Status, e.Message)
}

// Sentinel errors. Pair with errors.Is.
var (
	ErrNotFound     = errors.New("strapi: not found")
	ErrUnauthorized = errors.New("strapi: unauthorized")
	ErrForbidden    = errors.New("strapi: forbidden")
	ErrValidation   = errors.New("strapi: validation error")
	ErrBadResponse  = errors.New("strapi: bad response")
)

func (e *Error) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.Status == 404 || e.Name == "NotFoundError"
	case ErrUnauthorized:
		return e.Status == 401 || e.Name == "UnauthorizedError"
	case ErrForbidden:
		return e.Status == 403 || e.Name == "ForbiddenError"
	case ErrValidation:
		return e.Status == 400 || e.Name == "ValidationError"
	}
	return false
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./... -run TestError -v`
Expected: PASS for all three subtests/cases.

- [ ] **Step 5: Commit**

```bash
git add errors.go errors_test.go
git commit -m "feat: add Error type with sentinel errors and errors.Is support"
```

---

## Task 3: Document envelope and List wrapper

**Files:**
- Create: `document.go`
- Create: `document_test.go`
- Create: `testdata/page_single.json`
- Create: `testdata/page_list.json`

Strapi v5 returns entries as flat objects with system fields (`id`, `documentId`, `createdAt`, `updatedAt`, `publishedAt`, `locale`) mixed in with user-defined fields. We model this with a generic `Document[T]` whose `UnmarshalJSON` does a two-pass decode: once into a shadow struct for the system fields, once directly into `Attributes T`. The user's `T` struct only needs json tags for its own fields.

The wrapper envelope for a single entry (`{"data": {...}, "meta": {}}`) is kept unexported (`single[T]`) — consumers always interact with `*Document[T]`, never the envelope directly. `List[T]` stays exported because consumers DO read it.

- [ ] **Step 1: Add the JSON fixtures**

Create `testdata/page_single.json`:

```json
{
  "data": {
    "id": 1,
    "documentId": "abc123documentidulid0001",
    "title": "Home",
    "slug": "home",
    "createdAt": "2026-01-02T03:04:05.000Z",
    "updatedAt": "2026-01-03T03:04:05.000Z",
    "publishedAt": "2026-01-02T03:04:05.000Z",
    "locale": "en"
  },
  "meta": {}
}
```

Create `testdata/page_list.json`:

```json
{
  "data": [
    {
      "id": 1,
      "documentId": "abc123documentidulid0001",
      "title": "Home",
      "slug": "home",
      "createdAt": "2026-01-02T03:04:05.000Z",
      "updatedAt": "2026-01-03T03:04:05.000Z",
      "publishedAt": "2026-01-02T03:04:05.000Z",
      "locale": "en"
    },
    {
      "id": 2,
      "documentId": "abc123documentidulid0002",
      "title": "About",
      "slug": "about",
      "createdAt": "2026-01-02T03:04:05.000Z",
      "updatedAt": "2026-01-03T03:04:05.000Z",
      "publishedAt": null,
      "locale": "en"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "pageSize": 25,
      "pageCount": 1,
      "total": 2
    }
  }
}
```

- [ ] **Step 2: Write the failing test**

Create `document_test.go`:

```go
package strapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type pageAttrs struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestDocumentUnmarshalSingle(t *testing.T) {
	raw := readFixture(t, "page_single.json")
	var env struct {
		Data Document[pageAttrs] `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	d := env.Data
	if d.ID != 1 {
		t.Errorf("ID = %d want 1", d.ID)
	}
	if d.DocumentID != "abc123documentidulid0001" {
		t.Errorf("DocumentID = %q", d.DocumentID)
	}
	if d.Locale != "en" {
		t.Errorf("Locale = %q", d.Locale)
	}
	if d.Attributes.Title != "Home" {
		t.Errorf("Title = %q", d.Attributes.Title)
	}
	if d.Attributes.Slug != "home" {
		t.Errorf("Slug = %q", d.Attributes.Slug)
	}
	wantCreated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if !d.CreatedAt.Equal(wantCreated) {
		t.Errorf("CreatedAt = %v want %v", d.CreatedAt, wantCreated)
	}
	if d.PublishedAt == nil {
		t.Fatalf("PublishedAt is nil")
	}
}

func TestDocumentUnmarshalListWithNullPublishedAt(t *testing.T) {
	raw := readFixture(t, "page_list.json")
	var env List[pageAttrs]
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("Data len = %d want 2", len(env.Data))
	}
	if env.Data[0].Attributes.Title != "Home" {
		t.Errorf("Data[0].Title = %q", env.Data[0].Attributes.Title)
	}
	if env.Data[1].PublishedAt != nil {
		t.Errorf("Data[1].PublishedAt should be nil (draft)")
	}
	if env.Meta.Pagination.Total != 2 {
		t.Errorf("Total = %d want 2", env.Meta.Pagination.Total)
	}
}
```

- [ ] **Step 3: Run the test and verify it fails**

Run: `go test ./... -run TestDocument -v`
Expected: compile error — `Document`, `List`, etc. undefined.

- [ ] **Step 4: Implement `document.go`**

Create `document.go`:

```go
package strapi

import (
	"encoding/json"
	"time"
)

// Document is the generic envelope for a single Strapi v5 entry. System
// fields (ID, DocumentID, timestamps, locale) are split out from user-defined
// content fields, which live in Attributes. T should be a struct whose json
// tags match the content type's field names.
type Document[T any] struct {
	ID          int        `json:"-"`
	DocumentID  string     `json:"-"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	PublishedAt *time.Time `json:"-"`
	Locale      string     `json:"-"`
	Attributes  T          `json:"-"`
}

// UnmarshalJSON splits Strapi v5's flat entry shape into system fields and
// user-defined attributes by running two decodes over the same payload. The
// system-fields decode is keyed by Strapi's well-known names; the attribute
// decode targets T directly, which sees only its own tagged fields.
func (d *Document[T]) UnmarshalJSON(data []byte) error {
	var sys struct {
		ID          int        `json:"id"`
		DocumentID  string     `json:"documentId"`
		CreatedAt   time.Time  `json:"createdAt"`
		UpdatedAt   time.Time  `json:"updatedAt"`
		PublishedAt *time.Time `json:"publishedAt"`
		Locale      string     `json:"locale"`
	}
	if err := json.Unmarshal(data, &sys); err != nil {
		return err
	}
	d.ID = sys.ID
	d.DocumentID = sys.DocumentID
	d.CreatedAt = sys.CreatedAt
	d.UpdatedAt = sys.UpdatedAt
	d.PublishedAt = sys.PublishedAt
	d.Locale = sys.Locale
	return json.Unmarshal(data, &d.Attributes)
}

// List is the envelope for collection-list responses.
type List[T any] struct {
	Data []Document[T] `json:"data"`
	Meta Meta          `json:"meta"`
}

// single is the unexported envelope for find-one / mutation responses.
// Consumers always get back *Document[T] from the public API.
type single[T any] struct {
	Data Document[T] `json:"data"`
	Meta Meta        `json:"meta"`
}

// Meta carries pagination and other response metadata.
type Meta struct {
	Pagination Pagination `json:"pagination"`
}

// Pagination is the page-based pagination metadata. Strapi also supports
// start/limit (offset) pagination; in that mode PageCount is zero and Total
// is still populated.
type Pagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"pageSize"`
	PageCount int `json:"pageCount"`
	Total     int `json:"total"`
}
```

- [ ] **Step 5: Run the test and verify it passes**

Run: `go test ./... -run TestDocument -v`
Expected: PASS for both `TestDocumentUnmarshalSingle` and `TestDocumentUnmarshalListWithNullPublishedAt`.

- [ ] **Step 6: Commit**

```bash
git add document.go document_test.go testdata/
git commit -m "feat: add generic Document[T], List[T] response envelopes"
```

---

## Task 4: Query encoder (qs-compatible bracket notation)

**Files:**
- Create: `query/encode.go`
- Create: `query/encode_test.go`

Strapi expects the JS `qs` library's bracket encoding (`populate[author][populate][0]=company`), NOT Go's default `net/url.Values` (which sorts keys and uses repeated `key=v1&key=v2`). We build a small ordered encoder that takes nested key paths and renders them in stable order.

- [ ] **Step 1: Write the failing test**

Create `query/encode_test.go`:

```go
package query

import "testing"

func TestEncoderFlat(t *testing.T) {
	var e encoder
	e.add([]string{"sort"}, "title:asc")
	e.add([]string{"locale"}, "en")
	got := e.String()
	want := "sort=title%3Aasc&locale=en"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderNested(t *testing.T) {
	var e encoder
	e.add([]string{"filters", "title", "$eq"}, "Hello")
	got := e.String()
	want := "filters%5Btitle%5D%5B%24eq%5D=Hello"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderIndexedArray(t *testing.T) {
	var e encoder
	e.add([]string{"populate", "0"}, "author")
	e.add([]string{"populate", "1"}, "cover")
	got := e.String()
	want := "populate%5B0%5D=author&populate%5B1%5D=cover"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderPreservesInsertionOrder(t *testing.T) {
	var e encoder
	e.add([]string{"z"}, "1")
	e.add([]string{"a"}, "2")
	got := e.String()
	want := "z=1&a=2"
	if got != want {
		t.Fatalf("got %q want %q (must NOT be alphabetized)", got, want)
	}
}

func TestEncoderEmpty(t *testing.T) {
	var e encoder
	if e.String() != "" {
		t.Fatalf("empty encoder should produce empty string")
	}
}

func TestEncoderURLEncodesValueNotBrackets(t *testing.T) {
	var e encoder
	e.add([]string{"filters", "name", "$contains"}, "a b/c")
	got := e.String()
	// brackets percent-encoded, value space → +, slash → %2F
	want := "filters%5Bname%5D%5B%24contains%5D=a+b%2Fc"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./query/... -run TestEncoder -v`
Expected: compile error — `encoder` undefined.

- [ ] **Step 3: Implement `query/encode.go`**

Create `query/encode.go`:

```go
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
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./query/... -run TestEncoder -v`
Expected: PASS for all six subtests.

- [ ] **Step 5: Commit**

```bash
git add query/encode.go query/encode_test.go
git commit -m "feat(query): add qs-compatible bracket encoder"
```

---

## Task 5: Query — basic options (sort, pagination, fields, locale, status)

**Files:**
- Create: `query/query.go`
- Create: `query/query_test.go`

`Query` is the public type. `Option` is a function on `*Query`. The public `Build()` returns the query string (no leading `?`).

- [ ] **Step 1: Write the failing test**

Create `query/query_test.go`:

```go
package query

import "testing"

func TestQueryEmpty(t *testing.T) {
	q := New()
	if got := q.Build(); got != "" {
		t.Fatalf("empty query should be empty string, got %q", got)
	}
}

func TestQuerySort(t *testing.T) {
	q := New(Sort("title:asc"))
	want := "sort%5B0%5D=title%3Aasc"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryMultiSort(t *testing.T) {
	q := New(Sort("title:asc", "createdAt:desc"))
	want := "sort%5B0%5D=title%3Aasc&sort%5B1%5D=createdAt%3Adesc"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryPaginate(t *testing.T) {
	q := New(Paginate(2, 50))
	want := "pagination%5Bpage%5D=2&pagination%5BpageSize%5D=50"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryPaginateOffset(t *testing.T) {
	q := New(PaginateOffset(100, 25))
	want := "pagination%5Bstart%5D=100&pagination%5Blimit%5D=25"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryFields(t *testing.T) {
	q := New(Fields("title", "slug"))
	want := "fields%5B0%5D=title&fields%5B1%5D=slug"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryLocale(t *testing.T) {
	q := New(Locale("fr"))
	if got := q.Build(); got != "locale=fr" {
		t.Fatalf("got %q", got)
	}
}

func TestQueryStatusDraft(t *testing.T) {
	q := New(Status(StatusDraft))
	if got := q.Build(); got != "status=draft" {
		t.Fatalf("got %q", got)
	}
}

func TestQueryCombined(t *testing.T) {
	q := New(
		Paginate(1, 10),
		Sort("title:asc"),
		Locale("en"),
	)
	got := q.Build()
	want := "pagination%5Bpage%5D=1&pagination%5BpageSize%5D=10&sort%5B0%5D=title%3Aasc&locale=en"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./query/... -run TestQuery -v`
Expected: compile error — `New`, `Sort`, `Paginate`, etc. undefined.

- [ ] **Step 3: Implement `query/query.go`**

Create `query/query.go`:

```go
package query

import "strconv"

// Query accumulates Strapi REST query parameters. Build it via New(opts...)
// and serialize via Build, which returns a qs-compatible query string with
// no leading '?'.
type Query struct {
	enc encoder
}

// Option mutates a Query during construction.
type Option func(*Query)

// New builds a Query from the given options.
func New(opts ...Option) *Query {
	q := &Query{}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

// Build returns the encoded query string (no leading '?').
func (q *Query) Build() string {
	return q.enc.String()
}

// add is exposed to the package so filter.go and populate.go can write
// directly into the same ordered encoder.
func (q *Query) add(path []string, value string) {
	q.enc.add(path, value)
}

// Sort sorts results. Each entry is "field" or "field:asc"/"field:desc".
func Sort(fields ...string) Option {
	return func(q *Query) {
		for i, f := range fields {
			q.add([]string{"sort", strconv.Itoa(i)}, f)
		}
	}
}

// Paginate uses page-based pagination (Strapi default).
func Paginate(page, pageSize int) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "page"}, strconv.Itoa(page))
		q.add([]string{"pagination", "pageSize"}, strconv.Itoa(pageSize))
	}
}

// PaginateOffset uses offset-based pagination. Cannot be combined with
// Paginate — Strapi rejects mixed modes.
func PaginateOffset(start, limit int) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "start"}, strconv.Itoa(start))
		q.add([]string{"pagination", "limit"}, strconv.Itoa(limit))
	}
}

// WithCount controls whether pagination metadata includes the total count.
// Default is true on Strapi's side; set false to skip the COUNT query.
func WithCount(b bool) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "withCount"}, strconv.FormatBool(b))
	}
}

// Fields selects which top-level fields are returned.
func Fields(names ...string) Option {
	return func(q *Query) {
		for i, n := range names {
			q.add([]string{"fields", strconv.Itoa(i)}, n)
		}
	}
}

// Locale restricts the query to a single locale. Use "all" for every locale.
func Locale(code string) Option {
	return func(q *Query) {
		q.add([]string{"locale"}, code)
	}
}

// PublicationStatus is the v5 replacement for v4's publicationState.
type PublicationStatus string

const (
	StatusDraft     PublicationStatus = "draft"
	StatusPublished PublicationStatus = "published"
)

// Status filters by draft/published state (v5).
func Status(s PublicationStatus) Option {
	return func(q *Query) {
		q.add([]string{"status"}, string(s))
	}
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./query/... -run TestQuery -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add query/query.go query/query_test.go
git commit -m "feat(query): add Query + basic options (sort, paginate, fields, locale, status)"
```

---

## Task 6: Query — filters (operators + logical combinators)

**Files:**
- Create: `query/filter.go`
- Create: `query/filter_test.go`

Filters are nested arbitrarily under `filters[...]`. A `Filter` is anything that knows how to encode itself given a path prefix. The encode signature lives in this package (uses the internal `encoder` via the `Query.add` callback).

- [ ] **Step 1: Write the failing test**

Create `query/filter_test.go`:

```go
package query

import "testing"

func TestFilterEq(t *testing.T) {
	q := New(Where(Eq("title", "Hello")))
	want := "filters%5Btitle%5D%5B%24eq%5D=Hello"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterIn(t *testing.T) {
	q := New(Where(In("id", 1, 2, 3)))
	want := "filters%5Bid%5D%5B%24in%5D%5B0%5D=1&filters%5Bid%5D%5B%24in%5D%5B1%5D=2&filters%5Bid%5D%5B%24in%5D%5B2%5D=3"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterNullNotNull(t *testing.T) {
	q := New(Where(Null("publishedAt")))
	if got := q.Build(); got != "filters%5BpublishedAt%5D%5B%24null%5D=true" {
		t.Fatalf("got %q", got)
	}
	q2 := New(Where(NotNull("publishedAt")))
	if got := q2.Build(); got != "filters%5BpublishedAt%5D%5B%24notNull%5D=true" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterBetween(t *testing.T) {
	q := New(Where(Between("views", 10, 100)))
	want := "filters%5Bviews%5D%5B%24between%5D%5B0%5D=10&filters%5Bviews%5D%5B%24between%5D%5B1%5D=100"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterAnd(t *testing.T) {
	q := New(Where(And(
		Eq("status", "active"),
		Gt("views", 100),
	)))
	want := "filters%5B%24and%5D%5B0%5D%5Bstatus%5D%5B%24eq%5D=active&filters%5B%24and%5D%5B1%5D%5Bviews%5D%5B%24gt%5D=100"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestFilterOrNestedAnd(t *testing.T) {
	q := New(Where(Or(
		Eq("featured", true),
		And(Eq("status", "published"), Gt("views", 1000)),
	)))
	want := "filters%5B%24or%5D%5B0%5D%5Bfeatured%5D%5B%24eq%5D=true&filters%5B%24or%5D%5B1%5D%5B%24and%5D%5B0%5D%5Bstatus%5D%5B%24eq%5D=published&filters%5B%24or%5D%5B1%5D%5B%24and%5D%5B1%5D%5Bviews%5D%5B%24gt%5D=1000"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestFilterContainsCaseInsensitive(t *testing.T) {
	q := New(Where(ContainsI("title", "hello")))
	if got := q.Build(); got != "filters%5Btitle%5D%5B%24containsi%5D=hello" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterNestedRelation(t *testing.T) {
	// filters on a relation path: filters[author][name][$eq]=John
	q := New(Where(EqPath([]string{"author", "name"}, "John")))
	want := "filters%5Bauthor%5D%5Bname%5D%5B%24eq%5D=John"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./query/... -run TestFilter -v`
Expected: compile error — `Where`, `Eq`, `And`, etc. undefined.

- [ ] **Step 3: Implement `query/filter.go`**

Create `query/filter.go`:

```go
package query

import (
	"fmt"
	"strconv"
)

// Filter is anything that can encode itself into a Query at a given prefix.
// All path components are appended literally — callers must include array
// indices for $and/$or children themselves (Or/And handle this).
type Filter interface {
	encode(q *Query, prefix []string)
}

// Where adds a top-level filter under filters[...].
func Where(f Filter) Option {
	return func(q *Query) {
		f.encode(q, []string{"filters"})
	}
}

// --- comparison operators ------------------------------------------------

type cmp struct {
	path []string
	op   string
	val  any
}

func (c cmp) encode(q *Query, prefix []string) {
	full := append(append([]string{}, prefix...), c.path...)
	full = append(full, c.op)
	q.add(full, fmt.Sprint(c.val))
}

// Eq matches when field equals val.
func Eq(field string, val any) Filter        { return cmp{[]string{field}, "$eq", val} }
func EqI(field string, val any) Filter       { return cmp{[]string{field}, "$eqi", val} }
func Ne(field string, val any) Filter        { return cmp{[]string{field}, "$ne", val} }
func Lt(field string, val any) Filter        { return cmp{[]string{field}, "$lt", val} }
func Lte(field string, val any) Filter       { return cmp{[]string{field}, "$lte", val} }
func Gt(field string, val any) Filter        { return cmp{[]string{field}, "$gt", val} }
func Gte(field string, val any) Filter       { return cmp{[]string{field}, "$gte", val} }
func Contains(field string, val any) Filter  { return cmp{[]string{field}, "$contains", val} }
func ContainsI(field string, val any) Filter { return cmp{[]string{field}, "$containsi", val} }
func NotContains(field string, val any) Filter {
	return cmp{[]string{field}, "$notContains", val}
}
func StartsWith(field string, val any) Filter { return cmp{[]string{field}, "$startsWith", val} }
func EndsWith(field string, val any) Filter   { return cmp{[]string{field}, "$endsWith", val} }

// EqPath is Eq across a dotted/nested field path (filters on a relation).
func EqPath(path []string, val any) Filter {
	return cmp{path, "$eq", val}
}

// --- null / range / set --------------------------------------------------

// Null matches entries where the field is null.
func Null(field string) Filter {
	return cmp{[]string{field}, "$null", true}
}

// NotNull matches entries where the field is not null.
func NotNull(field string) Filter {
	return cmp{[]string{field}, "$notNull", true}
}

// Between matches values in the inclusive range [lo, hi].
func Between(field string, lo, hi any) Filter {
	return arrayFilter{field: field, op: "$between", vals: []any{lo, hi}}
}

// In matches values in the given set.
func In(field string, vals ...any) Filter {
	return arrayFilter{field: field, op: "$in", vals: vals}
}

// NotIn matches values NOT in the given set.
func NotIn(field string, vals ...any) Filter {
	return arrayFilter{field: field, op: "$notIn", vals: vals}
}

type arrayFilter struct {
	field string
	op    string
	vals  []any
}

func (a arrayFilter) encode(q *Query, prefix []string) {
	for i, v := range a.vals {
		full := append(append([]string{}, prefix...), a.field, a.op, strconv.Itoa(i))
		q.add(full, fmt.Sprint(v))
	}
}

// --- logical combinators -------------------------------------------------

type logical struct {
	op       string // "$and", "$or", "$not"
	children []Filter
}

func (l logical) encode(q *Query, prefix []string) {
	if l.op == "$not" {
		full := append(append([]string{}, prefix...), "$not")
		if len(l.children) == 1 {
			l.children[0].encode(q, full)
		}
		return
	}
	for i, ch := range l.children {
		full := append(append([]string{}, prefix...), l.op, strconv.Itoa(i))
		ch.encode(q, full)
	}
}

// And combines filters with logical AND.
func And(filters ...Filter) Filter {
	return logical{op: "$and", children: filters}
}

// Or combines filters with logical OR.
func Or(filters ...Filter) Filter {
	return logical{op: "$or", children: filters}
}

// Not negates a filter.
func Not(f Filter) Filter {
	return logical{op: "$not", children: []Filter{f}}
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./query/... -run TestFilter -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add query/filter.go query/filter_test.go
git commit -m "feat(query): add filter operators and logical combinators"
```

---

## Task 7: Query — populate (simple + builder for deep relations)

**Files:**
- Create: `query/populate.go`
- Create: `query/populate_test.go`

Three populate shapes to support:
1. `populate=*` — populate all top-level relations
2. `populate[0]=a&populate[1]=b` — list of named relations
3. Deep populate via builder: `populate[author][fields][0]=name&populate[author][populate][0]=company`

- [ ] **Step 1: Write the failing test**

Create `query/populate_test.go`:

```go
package query

import "testing"

func TestPopulateAll(t *testing.T) {
	q := New(PopulateAll())
	if got := q.Build(); got != "populate=%2A" {
		t.Fatalf("got %q", got)
	}
}

func TestPopulateNamedList(t *testing.T) {
	q := New(Populate("author", "cover"))
	want := "populate%5B0%5D=author&populate%5B1%5D=cover"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPopulateFieldFields(t *testing.T) {
	q := New(With(
		Field("author").Fields("name", "email"),
	))
	want := "populate%5Bauthor%5D%5Bfields%5D%5B0%5D=name&populate%5Bauthor%5D%5Bfields%5D%5B1%5D=email"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestPopulateDeep(t *testing.T) {
	q := New(With(
		Field("articles").
			Sort("publishedAt:desc").
			Populate(Field("category").Fields("name")),
	))
	want := "populate%5Barticles%5D%5Bsort%5D%5B0%5D=publishedAt%3Adesc&populate%5Barticles%5D%5Bpopulate%5D%5Bcategory%5D%5Bfields%5D%5B0%5D=name"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestPopulateWithFilter(t *testing.T) {
	q := New(With(
		Field("comments").Where(Eq("approved", true)),
	))
	want := "populate%5Bcomments%5D%5Bfilters%5D%5Bapproved%5D%5B%24eq%5D=true"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./query/... -run TestPopulate -v`
Expected: compile error — `PopulateAll`, `Populate`, `With`, `Field`, etc. undefined.

- [ ] **Step 3: Implement `query/populate.go`**

Create `query/populate.go`:

```go
package query

import "strconv"

// PopulateAll adds populate=* (shallow — one level).
func PopulateAll() Option {
	return func(q *Query) {
		q.add([]string{"populate"}, "*")
	}
}

// Populate adds populate[0]=name&populate[1]=other for the named relations.
func Populate(fields ...string) Option {
	return func(q *Query) {
		for i, f := range fields {
			q.add([]string{"populate", strconv.Itoa(i)}, f)
		}
	}
}

// With adds one or more deep-populate clauses built via Field.
func With(builders ...*PopulateBuilder) Option {
	return func(q *Query) {
		for _, b := range builders {
			b.encode(q, []string{"populate"})
		}
	}
}

// Field starts a deep-populate clause for the given relation field.
// Chain Fields, Sort, Where, Populate to refine it.
//
// Example:
//
//	Field("articles").
//	    Sort("publishedAt:desc").
//	    Populate(Field("author").Fields("name"))
type PopulateBuilder struct {
	name    string
	fields  []string
	sort    []string
	filter  Filter
	nested  []*PopulateBuilder
}

// Field constructs a populate builder for the named relation.
func Field(name string) *PopulateBuilder {
	return &PopulateBuilder{name: name}
}

// Fields selects which sub-fields of the populated relation to return.
func (p *PopulateBuilder) Fields(names ...string) *PopulateBuilder {
	p.fields = append(p.fields, names...)
	return p
}

// Sort sets the sort order on the populated relation.
func (p *PopulateBuilder) Sort(specs ...string) *PopulateBuilder {
	p.sort = append(p.sort, specs...)
	return p
}

// Where applies a filter to the populated relation.
func (p *PopulateBuilder) Where(f Filter) *PopulateBuilder {
	p.filter = f
	return p
}

// Populate nests further populate builders inside this one.
func (p *PopulateBuilder) Populate(children ...*PopulateBuilder) *PopulateBuilder {
	p.nested = append(p.nested, children...)
	return p
}

func (p *PopulateBuilder) encode(q *Query, prefix []string) {
	base := append(append([]string{}, prefix...), p.name)
	for i, f := range p.fields {
		q.add(append(append([]string{}, base...), "fields", strconv.Itoa(i)), f)
	}
	for i, s := range p.sort {
		q.add(append(append([]string{}, base...), "sort", strconv.Itoa(i)), s)
	}
	if p.filter != nil {
		p.filter.encode(q, append(append([]string{}, base...), "filters"))
	}
	for _, child := range p.nested {
		child.encode(q, append(append([]string{}, base...), "populate"))
	}
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./query/... -run TestPopulate -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Run the whole query package test suite**

Run: `go test ./query/... -v`
Expected: PASS for every test in the package.

- [ ] **Step 6: Commit**

```bash
git add query/populate.go query/populate_test.go
git commit -m "feat(query): add populate and deep-populate builder"
```

---

## Task 8: Client + functional options

**Files:**
- Create: `client.go`
- Create: `options.go`
- Create: `client_test.go`

- [ ] **Step 1: Write the failing test**

Create `client_test.go`:

```go
package strapi

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := New(WithBaseURL("https://cms.example.com"))
	if c.BaseURL() != "https://cms.example.com" {
		t.Errorf("BaseURL = %q", c.BaseURL())
	}
	if c.HTTPClient() == nil {
		t.Error("HTTPClient should default to non-nil")
	}
	if c.HTTPClient().Timeout == 0 {
		t.Error("default timeout should be non-zero")
	}
}

func TestNewClientStripsTrailingSlash(t *testing.T) {
	c := New(WithBaseURL("https://cms.example.com/"))
	if c.BaseURL() != "https://cms.example.com" {
		t.Errorf("BaseURL = %q want trailing slash stripped", c.BaseURL())
	}
}

func TestNewClientWithToken(t *testing.T) {
	c := New(WithBaseURL("https://x"), WithToken("secret"))
	if c.token != "secret" {
		t.Errorf("token = %q", c.token)
	}
}

func TestNewClientWithCustomHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 5 * time.Second}
	c := New(WithBaseURL("https://x"), WithHTTPClient(hc))
	if c.HTTPClient() != hc {
		t.Error("WithHTTPClient should set the http.Client")
	}
}

func TestNewClientPanicsWithoutBaseURL(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when base URL missing")
		}
	}()
	_ = New()
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test -run TestNewClient -v`
Expected: compile error — `New`, `WithBaseURL`, etc. undefined.

- [ ] **Step 3: Implement `options.go`**

Create `options.go`:

```go
package strapi

import "net/http"

// Option configures a Client during New.
type Option func(*Client)

// WithBaseURL sets the Strapi instance URL (e.g. "https://cms.example.com").
// Trailing slashes are stripped. Required.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		for len(url) > 0 && url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}
		c.baseURL = url
	}
}

// WithToken sets the Strapi API token sent as `Authorization: Bearer <token>`.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.httpClient = h
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}
```

- [ ] **Step 4: Implement `client.go`**

Create `client.go`:

```go
package strapi

import (
	"net/http"
	"time"
)

const defaultUserAgent = "strapi-go/0.1 (+https://github.com/jorgejr568/strapi-go)"

// Client is the entry point to the Strapi API. Construct it with New and
// pass it to Collection[T] / SingleType[T] or the top-level helpers.
type Client struct {
	baseURL    string
	token      string
	userAgent  string
	httpClient *http.Client
}

// New constructs a Client. WithBaseURL is required; calling New without it
// panics.
func New(opts ...Option) *Client {
	c := &Client{
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.baseURL == "" {
		panic("strapi: WithBaseURL is required")
	}
	return c
}

// BaseURL returns the configured Strapi base URL (without trailing slash).
func (c *Client) BaseURL() string { return c.baseURL }

// HTTPClient returns the underlying *http.Client.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }
```

- [ ] **Step 5: Run the test and verify it passes**

Run: `go test -run TestNewClient -v`
Expected: PASS for all five subtests.

- [ ] **Step 6: Commit**

```bash
git add client.go options.go client_test.go
git commit -m "feat: add Client with functional options"
```

---

## Task 9: HTTP request execution (body-aware)

**Files:**
- Create: `do.go`
- Create: `do_test.go`
- Create: `testdata/error_404.json`

The `do` method is the single internal helper for every HTTP request the SDK makes. It accepts the method, path, raw query string, optional JSON body to marshal, and optional decode target. Reads pass `body=nil`; writes pass a struct/map. Deletes may also pass `dst=nil` to discard the response.

- [ ] **Step 1: Add the error fixture**

Create `testdata/error_404.json`:

```json
{
  "data": null,
  "error": {
    "status": 404,
    "name": "NotFoundError",
    "message": "Not Found",
    "details": {}
  }
}
```

- [ ] **Step 2: Write the failing test**

Create `do_test.go`:

```go
package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestDoSuccessDecodesGET(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q", r.Method)
		}
		if r.URL.Path != "/api/pages/abc" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer t0k" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "strapi-go/") {
			t.Errorf("User-Agent = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		b, _ := os.ReadFile(filepath.Join("testdata", "page_single.json"))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL), WithToken("t0k"))
	var out single[pageAttrs]
	err := c.do(context.Background(), http.MethodGet, "/api/pages/abc", "", nil, &out)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if out.Data.Attributes.Title != "Home" {
		t.Errorf("Title = %q", out.Data.Attributes.Title)
	}
}

func TestDoMarshalsBodyAndSetsContentType(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		b, _ := io.ReadAll(r.Body)
		var got map[string]any
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("body json: %v", err)
		}
		data, _ := got["data"].(map[string]any)
		if data["title"] != "New" {
			t.Errorf("data.title = %v", data["title"])
		}
		w.Write([]byte(`{"data":{"id":1,"documentId":"new","title":"New","slug":"new","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	body := map[string]any{"data": map[string]any{"title": "New", "slug": "new"}}
	var out single[pageAttrs]
	if err := c.do(context.Background(), http.MethodPost, "/api/pages", "", body, &out); err != nil {
		t.Fatal(err)
	}
}

func TestDoErrorDecodes404(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		b, _ := os.ReadFile(filepath.Join("testdata", "error_404.json"))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL))
	var out single[pageAttrs]
	err := c.do(context.Background(), http.MethodGet, "/api/pages/missing", "", nil, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, expected errors.Is(ErrNotFound)", err)
	}
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("err = %v, expected errors.As(*Error)", err)
	}
	if se.Name != "NotFoundError" {
		t.Errorf("Name = %q", se.Name)
	}
}

func TestDoNoAuthHeaderWhenTokenAbsent(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization should be empty, got %q", got)
		}
		w.Write([]byte(`{"data":{"id":1,"documentId":"x","title":"a","slug":"a","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	})
	c := New(WithBaseURL(srv.URL))
	var out single[pageAttrs]
	if err := c.do(context.Background(), http.MethodGet, "/api/pages/1", "", nil, &out); err != nil {
		t.Fatal(err)
	}
}

func TestDoDiscardsResponseWhenDstNil(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204) // No Content
	})
	c := New(WithBaseURL(srv.URL))
	if err := c.do(context.Background(), http.MethodDelete, "/api/pages/abc", "", nil, nil); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Run the test and verify it fails**

Run: `go test -run TestDo -v`
Expected: compile error — `(*Client).do` undefined.

- [ ] **Step 4: Implement `do.go`**

Create `do.go`:

```go
package strapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// do builds and executes an HTTP request against the configured Strapi
// instance, decoding 2xx responses into dst and non-2xx responses into a
// typed *Error.
//
// Params:
//   - method:   "GET" | "POST" | "PUT" | "DELETE"
//   - path:     "/api/<endpoint>[/<documentId>]"
//   - rawQuery: query string without leading '?', or "" for none
//   - body:     any value to JSON-encode as the request body, or nil
//   - dst:      target for 2xx response decoding, or nil to discard
func (c *Client) do(ctx context.Context, method, path, rawQuery string, body, dst any) error {
	url := c.baseURL + path
	if rawQuery != "" {
		if strings.Contains(url, "?") {
			url += "&" + rawQuery
		} else {
			url += "?" + rawQuery
		}
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("strapi: marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("strapi: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("strapi: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("strapi: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp.StatusCode, respBody)
	}

	if dst == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		return fmt.Errorf("%w: %v", ErrBadResponse, err)
	}
	return nil
}

func decodeError(status int, body []byte) error {
	var envelope struct {
		Error *Error `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		if envelope.Error.Status == 0 {
			envelope.Error.Status = status
		}
		return envelope.Error
	}
	return &Error{
		Status:  status,
		Name:    "UnknownError",
		Message: strings.TrimSpace(string(body)),
	}
}
```

- [ ] **Step 5: Run the test and verify it passes**

Run: `go test -run TestDo -v`
Expected: PASS for all five subtests.

- [ ] **Step 6: Commit**

```bash
git add do.go do_test.go testdata/error_404.json
git commit -m "feat: add internal do() with body marshaling and typed error decoding"
```

---

## Task 10: Collection[T] — Find and List (reads)

**Files:**
- Create: `collection.go`
- Create: `collection_test.go`

This is the public ergonomic surface most consumers will use for reads. Two methods: `Find(ctx, documentID, opts...)` returns `*Document[T]`; `List(ctx, opts...)` returns `*List[T]`. Both accept variadic `query.Option`s. Mutations are added in Task 11.

Top-level generic helpers `Find[T]` / `List[T]` are also provided for one-shot usage.

- [ ] **Step 1: Write the failing test**

Create `collection_test.go`:

```go
package strapi

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgejr568/strapi-go/query"
)

func TestCollectionFind(t *testing.T) {
	var gotPath, gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		b, _ := os.ReadFile(filepath.Join("testdata", "page_single.json"))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")

	page, err := pages.Find(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if page.Attributes.Title != "Home" {
		t.Errorf("Title = %q", page.Attributes.Title)
	}
	if gotPath != "/api/pages/abc" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q want empty", gotQuery)
	}
}

func TestCollectionFindWithQuery(t *testing.T) {
	var gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		b, _ := os.ReadFile(filepath.Join("testdata", "page_single.json"))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")
	_, err := pages.Find(context.Background(), "abc",
		query.Locale("en"),
		query.Populate("author"),
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("locale") != "en" {
		t.Errorf("locale param missing: %v", parsed)
	}
	if parsed.Get("populate[0]") != "author" {
		t.Errorf("populate[0] missing: %v", parsed)
	}
}

func TestCollectionList(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		b, _ := os.ReadFile(filepath.Join("testdata", "page_list.json"))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")
	res, err := pages.List(context.Background(), query.Paginate(1, 25))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Data) != 2 {
		t.Fatalf("Data len = %d", len(res.Data))
	}
	if res.Data[0].Attributes.Title != "Home" {
		t.Errorf("Data[0].Title = %q", res.Data[0].Attributes.Title)
	}
	if res.Meta.Pagination.Total != 2 {
		t.Errorf("Total = %d", res.Meta.Pagination.Total)
	}
}

func TestTopLevelFindList(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var path string
		switch r.URL.Path {
		case "/api/pages/abc":
			path = "page_single.json"
		case "/api/pages":
			path = "page_list.json"
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		b, _ := os.ReadFile(filepath.Join("testdata", path))
		_, _ = w.Write(b)
	})

	c := New(WithBaseURL(srv.URL))
	page, err := Find[pageAttrs](context.Background(), c, "pages", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if page.Attributes.Title != "Home" {
		t.Errorf("Title = %q", page.Attributes.Title)
	}
	list, err := List[pageAttrs](context.Background(), c, "pages")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 2 {
		t.Fatalf("Data len = %d", len(list.Data))
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test -run TestCollection -v`
Expected: compile error — `Collection`, `NewCollection`, `Find`, `List` undefined.

- [ ] **Step 3: Implement `collection.go`**

Create `collection.go`:

```go
package strapi

import (
	"context"
	"net/http"

	"github.com/jorgejr568/strapi-go/query"
)

// Collection is a typed accessor for a Strapi collection-type endpoint.
// Construct it with NewCollection and reuse it across requests.
type Collection[T any] struct {
	client   *Client
	endpoint string // pluralApiId (e.g. "pages", "articles")
}

// NewCollection builds a Collection bound to the given endpoint
// (Strapi's pluralApiId, e.g. "pages").
func NewCollection[T any](c *Client, endpoint string) *Collection[T] {
	return &Collection[T]{client: c, endpoint: endpoint}
}

// Find fetches a single entry by its documentId (Strapi v5). Optional
// query options can populate relations, select fields, set locale, etc.
func (col *Collection[T]) Find(ctx context.Context, documentID string, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	if err := col.client.do(ctx, http.MethodGet, "/api/"+col.endpoint+"/"+documentID, q, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// List fetches a paginated list of entries. Use query options to filter,
// populate, sort, and paginate.
func (col *Collection[T]) List(ctx context.Context, opts ...query.Option) (*List[T], error) {
	q := query.New(opts...).Build()
	var env List[T]
	if err := col.client.do(ctx, http.MethodGet, "/api/"+col.endpoint, q, nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Find is a top-level convenience wrapper that builds a transient
// Collection[T] and calls Find.
func Find[T any](ctx context.Context, c *Client, endpoint, documentID string, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Find(ctx, documentID, opts...)
}

// List is a top-level convenience wrapper that builds a transient
// Collection[T] and calls List.
func List[T any](ctx context.Context, c *Client, endpoint string, opts ...query.Option) (*List[T], error) {
	return NewCollection[T](c, endpoint).List(ctx, opts...)
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test -run TestCollection -v`
Expected: PASS for all four tests.

Run: `go test ./...`
Expected: full suite green.

- [ ] **Step 5: Commit**

```bash
git add collection.go collection_test.go
git commit -m "feat: add Collection[T].Find/List + top-level helpers"
```

---

## Task 11: Collection[T] — Create, Update, Delete (mutations)

**Files:**
- Modify: `collection.go`
- Create: `collection_mutations_test.go`

Strapi expects mutation payloads wrapped under a `"data"` key: `{"data": {...attrs}}`. The Create method takes a typed `T` (the same struct used for reads). Update takes `any` so partial-update callers can pass `map[string]any{"title": "new"}` or a struct with pointer fields. Delete returns no payload — only `error`.

All three methods accept `query.Option`s (for example, `query.Locale("en")` to scope the mutation to a locale, or `query.Status(query.StatusDraft)` to act on the draft variant).

- [ ] **Step 1: Write the failing test**

Create `collection_mutations_test.go`:

```go
package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCollectionCreate(t *testing.T) {
	var gotMethod, gotPath, gotCT string
	var gotBody []byte
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"data":{"id":42,"documentId":"newdoc","title":"New","slug":"new","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")

	page, err := pages.Create(context.Background(), pageAttrs{Title: "New", Slug: "new"})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/api/pages" {
		t.Errorf("path = %q", gotPath)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q", gotCT)
	}

	var sent map[string]any
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatal(err)
	}
	data, _ := sent["data"].(map[string]any)
	if data == nil {
		t.Fatalf("body must wrap payload under \"data\": got %s", string(gotBody))
	}
	if data["title"] != "New" || data["slug"] != "new" {
		t.Errorf("data = %v", data)
	}

	if page.DocumentID != "newdoc" || page.Attributes.Title != "New" {
		t.Errorf("decoded = %+v", page)
	}
}

func TestCollectionUpdatePartialMap(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"data":{"id":1,"documentId":"existing","title":"Renamed","slug":"home","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")

	page, err := pages.Update(context.Background(), "existing",
		map[string]any{"title": "Renamed"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q want PUT", gotMethod)
	}
	if gotPath != "/api/pages/existing" {
		t.Errorf("path = %q", gotPath)
	}

	var sent map[string]any
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatal(err)
	}
	data, _ := sent["data"].(map[string]any)
	if data["title"] != "Renamed" {
		t.Errorf("data.title = %v", data["title"])
	}
	if _, ok := data["slug"]; ok {
		t.Errorf("partial update should not include slug, got %v", data)
	}

	if page.Attributes.Title != "Renamed" {
		t.Errorf("response title = %q", page.Attributes.Title)
	}
}

func TestCollectionDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")
	if err := pages.Delete(context.Background(), "existing"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q want DELETE", gotMethod)
	}
	if gotPath != "/api/pages/existing" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestCollectionCreateValidationError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write([]byte(`{"data":null,"error":{"status":400,"name":"ValidationError","message":"title is required","details":{"errors":[{"path":["title"],"message":"required"}]}}}`))
	})

	c := New(WithBaseURL(srv.URL))
	pages := NewCollection[pageAttrs](c, "pages")
	_, err := pages.Create(context.Background(), pageAttrs{Slug: "no-title"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
	var se *Error
	if !errors.As(err, &se) || !strings.Contains(se.Message, "title is required") {
		t.Errorf("error payload not surfaced: %v", err)
	}
}

func TestTopLevelMutationHelpers(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Write([]byte(`{"data":{"id":1,"documentId":"d","title":"t","slug":"s","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
		case http.MethodPut:
			w.Write([]byte(`{"data":{"id":1,"documentId":"d","title":"t2","slug":"s","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`))
		case http.MethodDelete:
			w.WriteHeader(204)
		}
	})
	c := New(WithBaseURL(srv.URL))

	if _, err := Create[pageAttrs](context.Background(), c, "pages", pageAttrs{Title: "t", Slug: "s"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update[pageAttrs](context.Background(), c, "pages", "d", map[string]any{"title": "t2"}); err != nil {
		t.Fatal(err)
	}
	if err := Delete(context.Background(), c, "pages", "d"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test -run TestCollection -v`
Expected: compile error — `(*Collection[T]).Create`, `Update`, `Delete`, top-level `Create`/`Update`/`Delete` all undefined.

- [ ] **Step 3: Extend `collection.go` with mutation methods**

Append the following to `collection.go` (after the existing top-level `List` helper):

```go
// Create inserts a new entry. attrs is the user-defined content for the
// entry (system fields are set by Strapi). The SDK wraps the payload as
// {"data": <attrs>} automatically. Optional query options apply to the
// returned representation (e.g. populate relations on the response).
func (col *Collection[T]) Create(ctx context.Context, attrs T, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := col.client.do(ctx, http.MethodPost, "/api/"+col.endpoint, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Update modifies an existing entry. attrs may be the full T or a partial
// payload (e.g. map[string]any{"title": "new"}) so callers can update
// individual fields without zeroing the rest. The SDK wraps the payload as
// {"data": <attrs>} automatically.
func (col *Collection[T]) Update(ctx context.Context, documentID string, attrs any, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := col.client.do(ctx, http.MethodPut, "/api/"+col.endpoint+"/"+documentID, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Delete removes an entry by documentId. The Strapi response body, if any,
// is discarded.
func (col *Collection[T]) Delete(ctx context.Context, documentID string) error {
	return col.client.do(ctx, http.MethodDelete, "/api/"+col.endpoint+"/"+documentID, "", nil, nil)
}

// Create is a top-level convenience wrapper.
func Create[T any](ctx context.Context, c *Client, endpoint string, attrs T, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Create(ctx, attrs, opts...)
}

// Update is a top-level convenience wrapper.
func Update[T any](ctx context.Context, c *Client, endpoint, documentID string, attrs any, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Update(ctx, documentID, attrs, opts...)
}

// Delete is a top-level convenience wrapper. The T type parameter is unused
// at the call site but kept for symmetry with the other helpers; if you
// don't care about T, pass `any`.
func Delete(ctx context.Context, c *Client, endpoint, documentID string) error {
	return (&Collection[any]{client: c, endpoint: endpoint}).Delete(ctx, documentID)
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test -run TestCollection -v`
Expected: PASS for all read AND mutation subtests (previous tests still green).

Run: `go test ./...`
Expected: full suite green.

- [ ] **Step 5: Commit**

```bash
git add collection.go collection_mutations_test.go
git commit -m "feat: add Collection[T].Create/Update/Delete + top-level mutation helpers"
```

---

## Task 12: SingleType[T] — single-type content accessor

**Files:**
- Create: `single_type.go`
- Create: `single_type_test.go`

Strapi single types (e.g. "Homepage", "Footer", "Global Settings") expose a single entry at `/api/:singularApiId` with no documentId segment. They support GET (read), PUT (update), and DELETE.

The accessor mirrors `Collection[T]` but without the documentId parameter and without `Create`/`List` (a single type has exactly one record). `Update` follows the same partial-payload pattern as `Collection.Update`.

- [ ] **Step 1: Write the failing test**

Create `single_type_test.go`:

```go
package strapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/jorgejr568/strapi-go/query"
)

type homepageAttrs struct {
	Headline string `json:"headline"`
	Subline  string `json:"subline,omitempty"`
}

func TestSingleTypeGet(t *testing.T) {
	var gotPath, gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"data":{"id":1,"documentId":"home1","headline":"Welcome","subline":"hi","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","publishedAt":"2026-01-01T00:00:00Z","locale":"en"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	hp := NewSingleType[homepageAttrs](c, "homepage")

	doc, err := hp.Get(context.Background(), query.Locale("en"))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/homepage" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery == "" {
		t.Errorf("query should include locale, got empty")
	}
	if doc.Attributes.Headline != "Welcome" {
		t.Errorf("Headline = %q", doc.Attributes.Headline)
	}
	if doc.DocumentID != "home1" {
		t.Errorf("DocumentID = %q", doc.DocumentID)
	}
}

func TestSingleTypeUpdate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"data":{"id":1,"documentId":"home1","headline":"Updated","subline":"hi","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	hp := NewSingleType[homepageAttrs](c, "homepage")

	doc, err := hp.Update(context.Background(), map[string]any{"headline": "Updated"})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/api/homepage" {
		t.Errorf("path = %q", gotPath)
	}

	var sent map[string]any
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatal(err)
	}
	data, _ := sent["data"].(map[string]any)
	if data["headline"] != "Updated" {
		t.Errorf("data = %v", data)
	}
	if doc.Attributes.Headline != "Updated" {
		t.Errorf("response headline = %q", doc.Attributes.Headline)
	}
}

func TestSingleTypeDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	c := New(WithBaseURL(srv.URL))
	hp := NewSingleType[homepageAttrs](c, "homepage")
	if err := hp.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/api/homepage" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestSingleTypeTopLevelHelpers(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"data":{"id":1,"documentId":"d","headline":"h","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
		case http.MethodPut:
			w.Write([]byte(`{"data":{"id":1,"documentId":"d","headline":"h2","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`))
		case http.MethodDelete:
			w.WriteHeader(204)
		}
	})
	c := New(WithBaseURL(srv.URL))

	if _, err := GetSingle[homepageAttrs](context.Background(), c, "homepage"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateSingle[homepageAttrs](context.Background(), c, "homepage", map[string]any{"headline": "h2"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSingle(context.Background(), c, "homepage"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test -run TestSingleType -v`
Expected: compile error — `SingleType`, `NewSingleType`, `GetSingle`, etc. undefined.

- [ ] **Step 3: Implement `single_type.go`**

Create `single_type.go`:

```go
package strapi

import (
	"context"
	"net/http"

	"github.com/jorgejr568/strapi-go/query"
)

// SingleType is a typed accessor for a Strapi single-type endpoint. A single
// type has exactly one record at /api/:singularApiId — there is no
// documentId in the path and no List operation.
type SingleType[T any] struct {
	client   *Client
	endpoint string // singularApiId (e.g. "homepage")
}

// NewSingleType builds a SingleType bound to the given endpoint
// (Strapi's singularApiId, e.g. "homepage").
func NewSingleType[T any](c *Client, endpoint string) *SingleType[T] {
	return &SingleType[T]{client: c, endpoint: endpoint}
}

// Get fetches the single-type record. Query options can populate relations,
// set locale, or select draft/published status.
func (s *SingleType[T]) Get(ctx context.Context, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	if err := s.client.do(ctx, http.MethodGet, "/api/"+s.endpoint, q, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Update modifies the single-type record. attrs may be the full T or a
// partial payload (e.g. map[string]any). The SDK wraps it as {"data": attrs}.
func (s *SingleType[T]) Update(ctx context.Context, attrs any, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := s.client.do(ctx, http.MethodPut, "/api/"+s.endpoint, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Delete clears the single-type record. Strapi semantics vary by version —
// some treat this as deleting the content row; others as resetting to
// empty. Either way, the response body is discarded.
func (s *SingleType[T]) Delete(ctx context.Context) error {
	return s.client.do(ctx, http.MethodDelete, "/api/"+s.endpoint, "", nil, nil)
}

// GetSingle is a top-level convenience wrapper.
func GetSingle[T any](ctx context.Context, c *Client, endpoint string, opts ...query.Option) (*Document[T], error) {
	return NewSingleType[T](c, endpoint).Get(ctx, opts...)
}

// UpdateSingle is a top-level convenience wrapper.
func UpdateSingle[T any](ctx context.Context, c *Client, endpoint string, attrs any, opts ...query.Option) (*Document[T], error) {
	return NewSingleType[T](c, endpoint).Update(ctx, attrs, opts...)
}

// DeleteSingle is a top-level convenience wrapper.
func DeleteSingle(ctx context.Context, c *Client, endpoint string) error {
	return (&SingleType[any]{client: c, endpoint: endpoint}).Delete(ctx)
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test -run TestSingleType -v`
Expected: PASS for all four subtests.

Run: `go test ./...`
Expected: full suite green.

- [ ] **Step 5: Commit**

```bash
git add single_type.go single_type_test.go
git commit -m "feat: add SingleType[T] with Get/Update/Delete + top-level helpers"
```

---

## Task 13: Media — File, Format types and ResolveURL helper

**Files:**
- Create: `media/media.go`
- Create: `media/media_test.go`

Strapi media files carry both a relative `url` (when the local provider serves them) and optional resized `formats`. Consumers should be able to (a) decode media without writing the verbose struct themselves and (b) resolve relative URLs against the Strapi base.

- [ ] **Step 1: Write the failing test**

Create `media/media_test.go`:

```go
package media

import (
	"encoding/json"
	"testing"
)

const sampleMedia = `{
  "id": 7,
  "documentId": "med123",
  "name": "pizza.jpg",
  "alternativeText": "A pizza",
  "caption": null,
  "width": 1920, "height": 1440,
  "hash": "abc", "ext": ".jpg", "mime": "image/jpeg", "size": 245.3,
  "url": "/uploads/pizza.jpg",
  "previewUrl": null, "provider": "local",
  "formats": {
    "thumbnail": { "name":"t","hash":"t","ext":".jpg","mime":"image/jpeg",
                    "width":245,"height":184,"size":12.5,"url":"/uploads/t.jpg" },
    "small":     { "name":"s","hash":"s","ext":".jpg","mime":"image/jpeg",
                    "width":500,"height":375,"size":35.1,"url":"/uploads/s.jpg" }
  },
  "createdAt":"2026-01-01T00:00:00.000Z",
  "updatedAt":"2026-01-01T00:00:00.000Z"
}`

func TestFileUnmarshal(t *testing.T) {
	var f File
	if err := json.Unmarshal([]byte(sampleMedia), &f); err != nil {
		t.Fatal(err)
	}
	if f.Name != "pizza.jpg" {
		t.Errorf("Name = %q", f.Name)
	}
	if f.URL != "/uploads/pizza.jpg" {
		t.Errorf("URL = %q", f.URL)
	}
	if f.Formats == nil || f.Formats.Thumbnail == nil {
		t.Fatalf("thumbnail format missing")
	}
	if f.Formats.Thumbnail.Width != 245 {
		t.Errorf("thumbnail width = %d", f.Formats.Thumbnail.Width)
	}
	if f.Formats.Large != nil {
		t.Errorf("large format should be absent (nil)")
	}
}

func TestResolveURLRelative(t *testing.T) {
	got := ResolveURL("https://cms.example.com", "/uploads/pizza.jpg")
	want := "https://cms.example.com/uploads/pizza.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveURLAbsolutePassthrough(t *testing.T) {
	abs := "https://cdn.example.com/u/x.jpg"
	if got := ResolveURL("https://cms.example.com", abs); got != abs {
		t.Errorf("got %q want %q", got, abs)
	}
}

func TestResolveURLBaseTrailingSlash(t *testing.T) {
	got := ResolveURL("https://cms.example.com/", "/uploads/x.jpg")
	want := "https://cms.example.com/uploads/x.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveURLEmptyURL(t *testing.T) {
	if got := ResolveURL("https://cms.example.com", ""); got != "" {
		t.Errorf("empty url should remain empty, got %q", got)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./media/... -v`
Expected: compile error — `File`, `ResolveURL` undefined.

- [ ] **Step 3: Implement `media/media.go`**

Create `media/media.go`:

```go
// Package media models Strapi media-library file entries and provides a
// helper to resolve relative URLs against a Strapi base URL.
package media

import (
	"strings"
	"time"
)

// File is a Strapi upload entry, returned either standalone via
// /api/upload or inlined as a populated relation.
type File struct {
	ID              int        `json:"id"`
	DocumentID      string     `json:"documentId,omitempty"`
	Name            string     `json:"name"`
	AlternativeText string     `json:"alternativeText,omitempty"`
	Caption         string     `json:"caption,omitempty"`
	Width           int        `json:"width,omitempty"`
	Height          int        `json:"height,omitempty"`
	Hash            string     `json:"hash"`
	Ext             string     `json:"ext"`
	MIME            string     `json:"mime"`
	Size            float64    `json:"size"`
	URL             string     `json:"url"`
	PreviewURL      string     `json:"previewUrl,omitempty"`
	Provider        string     `json:"provider"`
	Formats         *Formats   `json:"formats,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// Formats holds the set of resized variants Strapi may have generated.
// Each variant is optional — Strapi only emits a format when the source
// image is larger than the breakpoint.
type Formats struct {
	Thumbnail *Format `json:"thumbnail,omitempty"`
	Small     *Format `json:"small,omitempty"`
	Medium    *Format `json:"medium,omitempty"`
	Large     *Format `json:"large,omitempty"`
}

// Format describes a single resized image variant.
type Format struct {
	Name   string  `json:"name"`
	Hash   string  `json:"hash"`
	Ext    string  `json:"ext"`
	MIME   string  `json:"mime"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Size   float64 `json:"size"`
	URL    string  `json:"url"`
}

// ResolveURL joins a relative Strapi media URL against the base URL.
// Absolute URLs (http://, https://) and data URIs are returned unchanged.
// An empty url returns an empty string.
func ResolveURL(baseURL, url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "data:") {
		return url
	}
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	return baseURL + url
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./media/... -v`
Expected: PASS for all five subtests.

- [ ] **Step 5: Commit**

```bash
git add media/
git commit -m "feat(media): add File/Format types and ResolveURL helper"
```

---

## Task 14: Blocks — Rich-text AST + UnmarshalJSON dispatch

**Files:**
- Create: `blocks/blocks.go`
- Create: `blocks/blocks_test.go`

Strapi v4.13+/v5 rich-text blocks are an array of heterogeneous nodes (paragraph, heading, list, list-item, quote, code, link, image, text). Each node has a `type` discriminator. We model:
- `Blocks` — `[]Node` with custom `UnmarshalJSON` dispatch by `type`
- Concrete node types implementing a `Node` interface

- [ ] **Step 1: Write the failing test**

Create `blocks/blocks_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./blocks/... -v`
Expected: compile error — `Blocks`, `Paragraph`, etc. undefined.

- [ ] **Step 3: Implement `blocks/blocks.go`**

Create `blocks/blocks.go`:

```go
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
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./blocks/... -v`
Expected: PASS for both subtests.

- [ ] **Step 5: Commit**

```bash
git add blocks/blocks.go blocks/blocks_test.go
git commit -m "feat(blocks): add rich-text AST with discriminated UnmarshalJSON"
```

---

## Task 15: Blocks — HTML renderer

**Files:**
- Create: `blocks/html.go`
- Create: `blocks/html_test.go`

A simple HTML renderer is the most common immediate need — consumers can take the parsed Blocks and emit it in templates. Renderer is a plain function that returns a string; we use the `html` package for escaping.

- [ ] **Step 1: Write the failing test**

Create `blocks/html_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./blocks/... -run TestRenderHTML -v`
Expected: compile error — `RenderHTML` undefined.

- [ ] **Step 3: Implement `blocks/html.go`**

Create `blocks/html.go`:

```go
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
		writeTexts(sb, v.Children)
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
		writeTexts(sb, v.Children)
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
		writeTexts(sb, v.Children)
		sb.WriteString("</blockquote>")
	case *Code:
		sb.WriteString("<pre><code>")
		writeTexts(sb, v.Children)
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
```

- [ ] **Step 4: Run the test and verify it passes**

Run: `go test ./blocks/... -v`
Expected: PASS for all renderer subtests + previous AST tests.

- [ ] **Step 5: Commit**

```bash
git add blocks/html.go blocks/html_test.go
git commit -m "feat(blocks): add RenderHTML serializer"
```

---

## Task 16: Example program — fetch_pages

**Files:**
- Create: `examples/fetch_pages/main.go`

A runnable example showing the canonical use: define a `Page` struct, build a client, list pages, render the body blocks to HTML, resolve media URLs. Also demonstrates one mutation (an update) so consumers see the write path.

- [ ] **Step 1: Write the example**

Create `examples/fetch_pages/main.go`:

```go
// Example program: fetch Strapi Pages, render rich-text bodies to HTML,
// and resolve cover image URLs against the Strapi base URL. Demonstrates
// both reads and a mutation.
//
// Run:
//
//	STRAPI_URL=https://cms.example.com STRAPI_TOKEN=xxx go run ./examples/fetch_pages
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	strapi "github.com/jorgejr568/strapi-go"
	"github.com/jorgejr568/strapi-go/blocks"
	"github.com/jorgejr568/strapi-go/media"
	"github.com/jorgejr568/strapi-go/query"
)

// Page mirrors the user's "page" content type. Only user-defined fields
// need json tags; Strapi system fields (id, documentId, timestamps,
// locale) live on the surrounding strapi.Document[Page] envelope.
type Page struct {
	Title string         `json:"title"`
	Slug  string         `json:"slug"`
	Body  blocks.Blocks  `json:"body"`
	Cover *media.File    `json:"cover,omitempty"`
}

func main() {
	url := os.Getenv("STRAPI_URL")
	token := os.Getenv("STRAPI_TOKEN")
	if url == "" {
		log.Fatal("STRAPI_URL is required")
	}

	c := strapi.New(
		strapi.WithBaseURL(url),
		strapi.WithToken(token),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pages := strapi.NewCollection[Page](c, "pages")
	list, err := pages.List(ctx,
		query.Paginate(1, 10),
		query.Sort("title:asc"),
		query.Status(query.StatusPublished),
		query.With(
			query.Field("cover"),
		),
	)
	if err != nil {
		log.Fatalf("list: %v", err)
	}

	for _, p := range list.Data {
		fmt.Printf("== %s (%s) ==\n", p.Attributes.Title, p.DocumentID)
		if p.Attributes.Cover != nil {
			fmt.Printf("  cover: %s\n", media.ResolveURL(c.BaseURL(), p.Attributes.Cover.URL))
		}
		fmt.Println(blocks.RenderHTML(p.Attributes.Body))
	}

	// Mutation example: append " (updated)" to the first page's title.
	if len(list.Data) > 0 && os.Getenv("STRAPI_DEMO_WRITE") == "1" {
		first := list.Data[0]
		updated, err := pages.Update(ctx, first.DocumentID, map[string]any{
			"title": first.Attributes.Title + " (updated)",
		})
		if err != nil {
			log.Fatalf("update: %v", err)
		}
		fmt.Printf("updated: %s\n", updated.Attributes.Title)
	}
}
```

- [ ] **Step 2: Verify the example compiles**

Run: `go build ./examples/fetch_pages/`
Expected: exit 0, no output.

- [ ] **Step 3: Run vet on the whole module**

Run: `go vet ./...`
Expected: no findings.

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: all packages PASS.

- [ ] **Step 5: Commit**

```bash
git add examples/fetch_pages/main.go
git commit -m "docs: add fetch_pages example program with read + update demo"
```

---

## Task 17: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write the README**

Create `README.md`:

```markdown
# strapi-go

A Go SDK for the [Strapi](https://strapi.io) v5 REST API. Read and mutate
collection-type entries (Pages, Articles, …) and single-type records
(Homepage, Footer, …) as typed Go documents, with a fluent query builder,
parsed rich-text blocks, and resolved media URLs.

## Status

MVP. Supported today:

- Collection types: `Find` (by `documentId`), `List`, `Create`, `Update`, `Delete`
- Single types: `Get`, `Update`, `Delete`
- Query builder: pagination, sort, fields, locale, status, filters (all
  operators incl. `$and`/`$or`/`$not`), populate (simple + deep builder)
- Typed errors with `errors.Is(err, strapi.ErrNotFound)` etc.
- Rich-text **blocks** AST with an HTML renderer
- Media file types + `ResolveURL` helper

Not yet: file uploads, dynamic-zone typed registry, v4 response-format
compatibility, markdown renderer for blocks.

## Install

```bash
go get github.com/jorgejr568/strapi-go
```

Requires Go 1.22+.

## Quick start — read

```go
package main

import (
    "context"
    "fmt"
    "log"

    strapi "github.com/jorgejr568/strapi-go"
    "github.com/jorgejr568/strapi-go/blocks"
    "github.com/jorgejr568/strapi-go/media"
    "github.com/jorgejr568/strapi-go/query"
)

type Page struct {
    Title string        `json:"title"`
    Slug  string        `json:"slug"`
    Body  blocks.Blocks `json:"body"`
    Cover *media.File   `json:"cover,omitempty"`
}

func main() {
    c := strapi.New(
        strapi.WithBaseURL("https://cms.example.com"),
        strapi.WithToken("…"),
    )

    pages := strapi.NewCollection[Page](c, "pages")

    list, err := pages.List(context.Background(),
        query.Paginate(1, 10),
        query.Sort("title:asc"),
        query.With(query.Field("cover")),
    )
    if err != nil {
        log.Fatal(err)
    }

    for _, p := range list.Data {
        fmt.Println(p.Attributes.Title, "—", p.DocumentID)
        if p.Attributes.Cover != nil {
            fmt.Println("  cover:", media.ResolveURL(c.BaseURL(), p.Attributes.Cover.URL))
        }
        fmt.Println(blocks.RenderHTML(p.Attributes.Body))
    }
}
```

## Mutations

```go
pages := strapi.NewCollection[Page](c, "pages")

// Create — typed payload
created, _ := pages.Create(ctx, Page{Title: "New", Slug: "new"})

// Update — partial via map, or full via Page{…}
updated, _ := pages.Update(ctx, created.DocumentID, map[string]any{
    "title": "Renamed",
})

// Delete
_ = pages.Delete(ctx, created.DocumentID)
```

## Single types

```go
type Homepage struct {
    Headline string `json:"headline"`
    Subline  string `json:"subline,omitempty"`
}

hp := strapi.NewSingleType[Homepage](c, "homepage")

doc, _ := hp.Get(ctx, query.Locale("en"))
_, _ = hp.Update(ctx, map[string]any{"headline": "Welcome back"})
```

## Query builder

```go
import "github.com/jorgejr568/strapi-go/query"

q := []query.Option{
    query.Paginate(1, 25),
    query.Sort("publishedAt:desc"),
    query.Locale("en"),
    query.Status(query.StatusPublished),
    query.Where(query.And(
        query.Eq("status", "active"),
        query.Or(
            query.Eq("featured", true),
            query.Gt("views", 1000),
        ),
    )),
    query.With(
        query.Field("author").Fields("name", "email"),
        query.Field("categories").
            Sort("name:asc").
            Populate(query.Field("parent").Fields("name")),
    ),
}
```

## Errors

```go
_, err := pages.Find(ctx, "missing")
if errors.Is(err, strapi.ErrNotFound) { /* … */ }

var se *strapi.Error
if errors.As(err, &se) {
    fmt.Println(se.Status, se.Name, se.Message, string(se.Details))
}
```

## Rich-text blocks

`blocks.Blocks` decodes Strapi v4.13+ / v5 rich-text JSON into a typed
AST. Pass it to `blocks.RenderHTML` for a minimal escaped HTML
serialization, or walk it yourself for custom rendering.

## License

MIT
```

- [ ] **Step 2: Final check — everything builds and tests pass**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green, no findings.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add README with quick-start, mutations, single types, query builder"
```

---

## Self-Review

I re-checked the plan against the spec and against itself:

**Spec coverage:**
- "Fetch Pages and receive parsed documents" — Task 10 (Collection.Find/List) + Task 3 (Document[T]). ✅
- "Mutations (create/update/delete) — read-only client first" — explicitly in scope now, Task 11. ✅
- "Single-type endpoints (only collection types)" — explicitly in scope now, Task 12. ✅
- "Public layer with exported methods, interfaces" — every type/function in the plan is exported; module path is `github.com/jorgejr568/strapi-go`. ✅
- "Strapi REST API integration" — Tasks 5/6/7 build a Strapi-compatible `qs` query string with all major operators. ✅
- "Used by other projects" — examples/ + README + functional options Client. ✅
- "Generics" — `Document[T]`, `List[T]`, `Collection[T]`, `SingleType[T]`, top-level `Find[T]`/`List[T]`/`Create[T]`/`Update[T]`/`GetSingle[T]`/`UpdateSingle[T]`. ✅
- Strapi response shape: v5 native (flat fields). Tasks 3, 10, 11, 12 wire this through end-to-end. ✅

**Type consistency check:**
- `Document[T].Attributes` referenced consistently across Tasks 3, 10, 11, 12, 16, README. ✅
- `Document[T].DocumentID` consistent everywhere. ✅
- `NewCollection[T]` / `Collection[T].Find` / `List` / `Create` / `Update` / `Delete` — consistent across Tasks 10, 11, README, example. ✅
- `NewSingleType[T]` / `SingleType[T].Get` / `Update` / `Delete` — consistent across Tasks 12, README. ✅
- Top-level helpers: `Find[T]`, `List[T]` (Task 10), `Create[T]`, `Update[T]`, `Delete` (Task 11), `GetSingle[T]`, `UpdateSingle[T]`, `DeleteSingle` (Task 12). All present, all consistent. ✅
- `query.Where(...)` (filter entry point) used in Tasks 6 and README. ✅
- `query.With(...)` (populate entry point taking `*PopulateBuilder`) — used in Tasks 7, 16, README. ✅
- `query.Field(name)` builder — used identically in Tasks 7, 16, README. ✅
- `strapi.Error` (struct) and `strapi.Err*` sentinels — used identically across Tasks 2, 9, 11, README. ✅
- `media.File`, `media.ResolveURL` — Tasks 13, 16, README. ✅
- `blocks.Blocks`, `blocks.RenderHTML` — Tasks 14, 15, 16, README. ✅
- `single[T]` (unexported envelope) — defined in Task 3, used in Tasks 9, 10, 11, 12. ✅
- `(*Client).do` signature `(ctx, method, path, rawQuery, body, dst)` — used identically by Task 9 (test), Task 10 (read calls pass nil body), Task 11 (mutation calls pass body), Task 12 (single-type read/write/delete). ✅

**Placeholder scan:**
- No TBDs, no "implement later", no "add error handling", no "similar to Task N", no naked type references. Every code step has full code; every command has expected output. ✅

**One adjustment made inline:** The find-one envelope is now unexported (`single[T]`) since it's only ever an implementation detail — public methods return `*Document[T]`. This also frees the `Single` name space (we use `SingleType[T]` for the single-type accessor instead).

Plan is ready for execution.
