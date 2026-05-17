package strapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgejr568/strapi-go/media"
)

// authorAttrs is the content-type used by TestV4DecodesRelationsAndMedia.
type authorAttrs struct {
	Name   string      `json:"name"`
	Avatar *media.File `json:"avatar,omitempty"`
}

// tagAttrs is the content-type used by TestV4DecodesRelationsAndMedia.
type tagAttrs struct {
	Name string `json:"name"`
}

// postWithRelations is the content-type used by TestV4DecodesRelationsAndMedia.
type postWithRelations struct {
	Title  string                 `json:"title"`
	Slug   string                 `json:"slug"`
	Author *Document[authorAttrs] `json:"author,omitempty"`
	Tags   []Document[tagAttrs]   `json:"tags,omitempty"`
}

func serveV4Fixture(t *testing.T, name string) *Client {
	t.Helper()
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		_, _ = w.Write(b)
	})
	return New(WithBaseURL(srv.URL), WithAPIVersion(APIVersionV4))
}

func TestV4FindDecodesSingleEntry(t *testing.T) {
	c := serveV4Fixture(t, "page_single_v4.json")
	pages := NewCollection[pageAttrs](c, "pages")

	doc, err := pages.Find(context.Background(), "1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if doc.ID != 1 {
		t.Errorf("ID = %d want 1", doc.ID)
	}
	if doc.DocumentID != "" {
		t.Errorf("DocumentID should be empty in v4, got %q", doc.DocumentID)
	}
	if doc.Attributes.Title != "Home" || doc.Attributes.Slug != "home" {
		t.Errorf("attrs = %+v", doc.Attributes)
	}
	if doc.Locale != "en" {
		t.Errorf("Locale = %q", doc.Locale)
	}
}

func TestV4ListDecodesEntries(t *testing.T) {
	c := serveV4Fixture(t, "page_list_v4.json")
	pages := NewCollection[pageAttrs](c, "pages")

	list, err := pages.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 2 {
		t.Fatalf("Data len = %d want 2", len(list.Data))
	}
	if list.Data[0].Attributes.Title != "Home" {
		t.Errorf("Data[0].Title = %q", list.Data[0].Attributes.Title)
	}
	if list.Data[1].PublishedAt != nil {
		t.Errorf("Data[1].PublishedAt should be nil (draft)")
	}
	if list.Meta.Pagination.Total != 2 {
		t.Errorf("Total = %d want 2", list.Meta.Pagination.Total)
	}
}

func TestV4DecodesRelationsAndMedia(t *testing.T) {
	c := serveV4Fixture(t, "page_relations_v4.json")
	pages := NewCollection[postWithRelations](c, "pages")

	doc, err := pages.Find(context.Background(), "1")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	// Single relation envelope unwrapped → Author populated.
	if doc.Attributes.Author == nil {
		t.Fatal("Author is nil — single-relation envelope was not unwrapped")
	}
	if doc.Attributes.Author.ID != 5 {
		t.Errorf("Author.ID = %d want 5", doc.Attributes.Author.ID)
	}
	if doc.Attributes.Author.Attributes.Name != "Alice" {
		t.Errorf("Author.Name = %q want Alice", doc.Attributes.Author.Attributes.Name)
	}
	// Media nested inside relation → still resolves.
	avatar := doc.Attributes.Author.Attributes.Avatar
	if avatar == nil {
		t.Fatal("Avatar is nil — nested relation/media envelope not unwrapped")
	}
	if avatar.URL != "/uploads/alice.png" || avatar.MIME != "image/png" {
		t.Errorf("Avatar = %+v", avatar)
	}
	// Array relation envelope unwrapped → Tags has both items.
	if len(doc.Attributes.Tags) != 2 {
		t.Fatalf("Tags len = %d want 2", len(doc.Attributes.Tags))
	}
	if doc.Attributes.Tags[0].Attributes.Name != "go" || doc.Attributes.Tags[1].Attributes.Name != "rest" {
		t.Errorf("Tags = %+v", doc.Attributes.Tags)
	}
}
