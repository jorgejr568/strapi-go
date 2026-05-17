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
	var env ListResponse[pageAttrs]
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

func TestDocumentUnmarshalReturnsErrorOnInvalidJSON(t *testing.T) {
	var d Document[pageAttrs]
	err := d.UnmarshalJSON([]byte(`{"id": "not a number"}`))
	if err == nil {
		t.Fatal("expected error for invalid id type, got nil")
	}
}

func TestDocumentPathIDV5UsesDocumentID(t *testing.T) {
	d := Document[pageAttrs]{ID: 1, DocumentID: "doc-abc"}
	if got := d.PathID(); got != "doc-abc" {
		t.Errorf("PathID() = %q want %q", got, "doc-abc")
	}
}

func TestDocumentPathIDV4FallsBackToID(t *testing.T) {
	d := Document[pageAttrs]{ID: 42, DocumentID: ""}
	if got := d.PathID(); got != "42" {
		t.Errorf("PathID() = %q want %q", got, "42")
	}
}
