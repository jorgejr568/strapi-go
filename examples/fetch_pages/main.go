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
	Title string        `json:"title"`
	Slug  string        `json:"slug"`
	Body  blocks.Blocks `json:"body"`
	Cover *media.File   `json:"cover,omitempty"`
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
