package query

import "testing"

func TestFilterEq(t *testing.T) {
	q := New(Where(Eq("title", "Hello")))
	want := "filters%5Btitle%5D%5B%24eq%5D=Hello"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterIn(t *testing.T) {
	q := New(Where(In("id", 1, 2, 3)))
	want := "filters%5Bid%5D%5B%24in%5D%5B0%5D=1&filters%5Bid%5D%5B%24in%5D%5B1%5D=2&filters%5Bid%5D%5B%24in%5D%5B2%5D=3"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterNullNotNull(t *testing.T) {
	q := New(Where(Null("publishedAt")))
	if got := q.Build(); got != "filters%5BpublishedAt%5D%5B%24null%5D=true" {
		t.Fatalf("got %q", got)
	}
	q2 := New(Where(NotNull("publishedAt")))
	if got := q2.Build(); got != "filters%5BpublishedAt%5D%5B%24notNull%5D=true" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterBetween(t *testing.T) {
	q := New(Where(Between("views", 10, 100)))
	want := "filters%5Bviews%5D%5B%24between%5D%5B0%5D=10&filters%5Bviews%5D%5B%24between%5D%5B1%5D=100"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFilterAnd(t *testing.T) {
	q := New(Where(And(
		Eq("status", "active"),
		Gt("views", 100),
	)))
	want := "filters%5B%24and%5D%5B0%5D%5Bstatus%5D%5B%24eq%5D=active&filters%5B%24and%5D%5B1%5D%5Bviews%5D%5B%24gt%5D=100"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestFilterOrNestedAnd(t *testing.T) {
	q := New(Where(Or(
		Eq("featured", true),
		And(Eq("status", "published"), Gt("views", 1000)),
	)))
	want := "filters%5B%24or%5D%5B0%5D%5Bfeatured%5D%5B%24eq%5D=true&filters%5B%24or%5D%5B1%5D%5B%24and%5D%5B0%5D%5Bstatus%5D%5B%24eq%5D=published&filters%5B%24or%5D%5B1%5D%5B%24and%5D%5B1%5D%5Bviews%5D%5B%24gt%5D=1000"
	if got := q.Build(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestFilterContainsCaseInsensitive(t *testing.T) {
	q := New(Where(ContainsI("title", "hello")))
	if got := q.Build(); got != "filters%5Btitle%5D%5B%24containsi%5D=hello" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterNestedRelation(t *testing.T) {
	// filters on a relation path: filters[author][name][$eq]=John
	q := New(Where(EqPath([]string{"author", "name"}, "John")))
	want := "filters%5Bauthor%5D%5Bname%5D%5B%24eq%5D=John"
	if got := q.Build(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
