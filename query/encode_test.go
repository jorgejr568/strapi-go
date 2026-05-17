package query

import "testing"

func TestEncoderFlat(t *testing.T) {
	var e encoder
	e.add([]string{"sort"}, "title:asc")
	e.add([]string{"locale"}, "en")
	got := e.String()
	want := "sort=title%3Aasc&locale=en"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderNested(t *testing.T) {
	var e encoder
	e.add([]string{"filters", "title", "$eq"}, "Hello")
	got := e.String()
	want := "filters%5Btitle%5D%5B%24eq%5D=Hello"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderIndexedArray(t *testing.T) {
	var e encoder
	e.add([]string{"populate", "0"}, "author")
	e.add([]string{"populate", "1"}, "cover")
	got := e.String()
	want := "populate%5B0%5D=author&populate%5B1%5D=cover"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderPreservesInsertionOrder(t *testing.T) {
	var e encoder
	e.add([]string{"z"}, "1")
	e.add([]string{"a"}, "2")
	got := e.String()
	want := "z=1&a=2"
	if got != want {
		t.Fatalf("got %q want %q (must NOT be alphabetized)", got, want)
	}
}

func TestEncoderEmpty(t *testing.T) {
	var e encoder
	if e.String() != "" {
		t.Fatalf("empty encoder should produce empty string")
	}
}

func TestEncoderURLEncodesValueNotBrackets(t *testing.T) {
	var e encoder
	e.add([]string{"filters", "name", "$contains"}, "a b/c")
	got := e.String()
	// brackets percent-encoded, value space → +, slash → %2F
	want := "filters%5Bname%5D%5B%24contains%5D=a+b%2Fc"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestEncoderEmptyPathSegment(t *testing.T) {
	// An empty path is a no-op key (just emits `=value`) — guard exists in writeKey.
	var e encoder
	e.add([]string{}, "stray")
	got := e.String()
	if got != "=stray" {
		t.Fatalf("got %q want %q", got, "=stray")
	}
}
