package query

import "testing"

func TestQueryEmpty(t *testing.T) {
	q := New()
	if got := q.Build(); got != "" {
		t.Fatalf("empty query should be empty string, got %q", got)
	}
}

func TestQuerySort(t *testing.T) {
	q := New(Sort("title:asc"))
	want := "sort%5B0%5D=title%3Aasc"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryMultiSort(t *testing.T) {
	q := New(Sort("title:asc", "createdAt:desc"))
	want := "sort%5B0%5D=title%3Aasc&sort%5B1%5D=createdAt%3Adesc"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryPaginate(t *testing.T) {
	q := New(Paginate(2, 50))
	want := "pagination%5Bpage%5D=2&pagination%5BpageSize%5D=50"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryPaginateOffset(t *testing.T) {
	q := New(PaginateOffset(100, 25))
	want := "pagination%5Bstart%5D=100&pagination%5Blimit%5D=25"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryFields(t *testing.T) {
	q := New(Fields("title", "slug"))
	want := "fields%5B0%5D=title&fields%5B1%5D=slug"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryLocale(t *testing.T) {
	q := New(Locale("fr"))
	if got := q.Build(); got != "locale=fr" {
		t.Fatalf("got %q", got)
	}
}

func TestQueryStatusDraft(t *testing.T) {
	q := New(Status(StatusDraft))
	if got := q.Build(); got != "status=draft" {
		t.Fatalf("got %q", got)
	}
}

func TestQueryCombined(t *testing.T) {
	q := New(
		Paginate(1, 10),
		Sort("title:asc"),
		Locale("en"),
	)
	got := q.Build()
	want := "pagination%5Bpage%5D=1&pagination%5BpageSize%5D=10&sort%5B0%5D=title%3Aasc&locale=en"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
