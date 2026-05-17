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
