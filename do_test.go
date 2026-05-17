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

func TestDoErrorFallbackToUnknownErrorOnNonJSONBody(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Upstream proxy emits an HTML error page; SDK should synthesize
		// an UnknownError with the raw body as Message.
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(502)
		_, _ = w.Write([]byte("<html><body>502 Bad Gateway</body></html>"))
	})

	c := New(WithBaseURL(srv.URL))
	var out single[pageAttrs]
	err := c.do(context.Background(), http.MethodGet, "/api/pages/abc", "", nil, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("err = %v, expected *Error", err)
	}
	if se.Name != "UnknownError" {
		t.Errorf("Name = %q want UnknownError", se.Name)
	}
	if se.Status != 502 {
		t.Errorf("Status = %d want 502", se.Status)
	}
	if !strings.Contains(se.Message, "Bad Gateway") {
		t.Errorf("Message %q should contain raw body", se.Message)
	}
}

func TestDoErrorStatusBackfillWhenEnvelopeOmitsStatus(t *testing.T) {
	// Envelope decodes but has Status==0; SDK should backfill from HTTP code.
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"data":null,"error":{"name":"ServiceUnavailable","message":"db down"}}`))
	})

	c := New(WithBaseURL(srv.URL))
	var out single[pageAttrs]
	err := c.do(context.Background(), http.MethodGet, "/api/pages/abc", "", nil, &out)
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
	if se.Status != 503 {
		t.Errorf("Status = %d want 503 (backfilled from HTTP code)", se.Status)
	}
	if se.Name != "ServiceUnavailable" {
		t.Errorf("Name = %q", se.Name)
	}
}

func TestDoReturnsMarshalErrorForUnmarshallableBody(t *testing.T) {
	c := New(WithBaseURL("https://x"))
	// A chan cannot be JSON-marshaled.
	body := map[string]any{"data": make(chan int)}
	err := c.do(context.Background(), http.MethodPost, "/api/pages", "", body, nil)
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal body") {
		t.Errorf("err = %v, want 'marshal body' in message", err)
	}
}

func TestDoReturnsBuildRequestErrorForInvalidMethod(t *testing.T) {
	c := New(WithBaseURL("https://x"))
	// Method with a space is invalid per net/http.
	err := c.do(context.Background(), "BAD METHOD", "/api/pages", "", nil, nil)
	if err == nil {
		t.Fatal("expected build-request error")
	}
	if !strings.Contains(err.Error(), "build request") {
		t.Errorf("err = %v, want 'build request' in message", err)
	}
}

func TestDoReturnsHTTPErrorWhenServerUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so connection fails
	c := New(WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/api/pages/abc", "", nil, nil)
	if err == nil {
		t.Fatal("expected http error")
	}
	if !strings.Contains(err.Error(), "http") {
		t.Errorf("err = %v, want 'http' in message", err)
	}
}

func TestDoReturnsErrBadResponseOnInvalidJSON(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json at all"))
	})

	c := New(WithBaseURL(srv.URL))
	var out single[pageAttrs]
	err := c.do(context.Background(), http.MethodGet, "/api/pages/abc", "", nil, &out)
	if err == nil {
		t.Fatal("expected ErrBadResponse")
	}
	if !errors.Is(err, ErrBadResponse) {
		t.Errorf("err = %v, want errors.Is(ErrBadResponse)", err)
	}
}

func TestDoAppendsQueryWithAmpersandWhenPathHasQuestion(t *testing.T) {
	// rawQuery appending uses & when the URL already contains ?.
	// Build a path containing a literal ? so do() takes the & branch.
	var gotURL string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{}`))
	})
	c := New(WithBaseURL(srv.URL))
	err := c.do(context.Background(), http.MethodGet, "/api/pages?preset=foo", "locale=en", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "preset=foo&locale=en") {
		t.Errorf("URL = %q want '?preset=foo&locale=en'", gotURL)
	}
}

func TestDoV4ResponseIsNormalizedBeforeUnmarshal(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Strapi v4 single-entry shape.
		_, _ = w.Write([]byte(`{
			"data": {
				"id": 7,
				"attributes": {
					"title": "From v4",
					"slug":  "from-v4"
				}
			},
			"meta": {}
		}`))
	})

	c := New(WithBaseURL(srv.URL), WithAPIVersion(APIVersionV4))
	var out single[pageAttrs]
	if err := c.do(context.Background(), http.MethodGet, "/api/pages/7", "", nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if out.Data.ID != 7 {
		t.Errorf("ID = %d want 7", out.Data.ID)
	}
	if out.Data.Attributes.Title != "From v4" {
		t.Errorf("Title = %q want \"From v4\"", out.Data.Attributes.Title)
	}
	if out.Data.Attributes.Slug != "from-v4" {
		t.Errorf("Slug = %q", out.Data.Attributes.Slug)
	}
}
