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

Not yet:

- File uploads
- Dynamic-zone typed registry
- v4 response-format compatibility
- Markdown renderer for blocks
- Inline links and nested lists inside rich-text bodies are decoded as
  plain text only — the `Children` fields on inline-bearing block types
  are typed `[]Text` for MVP simplicity.

Note: the list envelope returned by `Collection[T].List` and the
top-level `List[T]` helper is `*ListResponse[T]`. (Go places types and
functions in the same package-scope namespace, so the envelope type and
the helper function cannot both be named `List`.) `list.Data` is still
`[]Document[T]`.

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

**Limitation:** the MVP types `Paragraph.Children`, `Heading.Children`,
`ListItem.Children`, `Quote.Children`, `Code.Children`, `Link.Children`,
and `Image.Children` are all `[]Text`. Non-Text inline nodes (e.g. an
inline link inside a paragraph, or a nested list inside a list-item) are
silently dropped during JSON decode. A future version will sealed-sum-type
the inline AST.

## Examples

A runnable end-to-end demo lives at
[`examples/fetch_pages/main.go`](examples/fetch_pages/main.go). It lists
published Pages, resolves their cover-image URLs, renders rich-text
bodies to HTML, and (optionally) performs a mutation. Run it with:

```bash
STRAPI_URL=https://cms.example.com STRAPI_TOKEN=xxx \
    go run ./examples/fetch_pages
```

Set `STRAPI_DEMO_WRITE=1` to also exercise the update path.

## License

MIT
