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

func TestResolveURLDataURIPassthrough(t *testing.T) {
	// data: URIs should pass through unchanged like absolute URLs.
	got := ResolveURL("https://cms.example.com", "data:image/png;base64,iVBORw0KGgo=")
	want := "data:image/png;base64,iVBORw0KGgo="
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveURLAddsLeadingSlash(t *testing.T) {
	// Relative URLs without a leading slash should get one prepended.
	got := ResolveURL("https://cms.example.com", "uploads/x.jpg")
	want := "https://cms.example.com/uploads/x.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestResolveURLBaseWithPathPrefix(t *testing.T) {
	// Base URL with a path prefix (common reverse-proxy deployment) preserves it.
	got := ResolveURL("https://example.com/strapi", "/uploads/x.jpg")
	want := "https://example.com/strapi/uploads/x.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
