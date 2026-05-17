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
