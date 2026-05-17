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
