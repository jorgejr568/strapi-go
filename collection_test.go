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
