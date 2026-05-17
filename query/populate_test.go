package query

import "testing"

func TestPopulateAll(t *testing.T) {
	q := New(PopulateAll())
	if got := q.Build(); got != "populate=%2A" {
		t.Fatalf("got %q", got)
	}
}

func TestPopulateNamedList(t *testing.T) {
	q := New(Populate("author", "cover"))
	want := "populate%5B0%5D=author&populate%5B1%5D=cover"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPopulateFieldFields(t *testing.T) {
	q := New(With(
		Field("author").Fields("name", "email"),
	))
	want := "populate%5Bauthor%5D%5Bfields%5D%5B0%5D=name&populate%5Bauthor%5D%5Bfields%5D%5B1%5D=email"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestPopulateDeep(t *testing.T) {
	q := New(With(
		Field("articles").
			Sort("publishedAt:desc").
			Populate(Field("category").Fields("name")),
	))
	want := "populate%5Barticles%5D%5Bsort%5D%5B0%5D=publishedAt%3Adesc&populate%5Barticles%5D%5Bpopulate%5D%5Bcategory%5D%5Bfields%5D%5B0%5D=name"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestPopulateWithFilter(t *testing.T) {
	q := New(With(
		Field("comments").Where(Eq("approved", true)),
	))
	want := "populate%5Bcomments%5D%5Bfilters%5D%5Bapproved%5D%5B%24eq%5D=true"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
