package strapi

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	e := &Error{Status: 404, Name: "NotFoundError", Message: "Not Found"}
	got := e.Error()
	want := "strapi: NotFoundError (404): Not Found"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestErrorIs(t *testing.T) {
	cases := []struct {
		name     string
		err      *Error
		sentinel error
		want     bool
	}{
		{"404 by status", &Error{Status: 404}, ErrNotFound, true},
		{"404 by name", &Error{Name: "NotFoundError"}, ErrNotFound, true},
		{"401 by status", &Error{Status: 401}, ErrUnauthorized, true},
		{"403 by status", &Error{Status: 403}, ErrForbidden, true},
		{"400 validation", &Error{Status: 400, Name: "ValidationError"}, ErrValidation, true},
		{"mismatch", &Error{Status: 500}, ErrNotFound, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.err, tc.sentinel); got != tc.want {
				t.Fatalf("errors.Is = %v want %v", got, tc.want)
			}
		})
	}
}

func TestErrorDetailsPreserved(t *testing.T) {
	raw := json.RawMessage(`{"errors":[{"path":["title"],"message":"required"}]}`)
	e := &Error{Status: 400, Name: "ValidationError", Message: "fail", Details: raw}
	if string(e.Details) != string(raw) {
		t.Fatalf("details not preserved")
	}
}

func TestErrorIsReturnsFalseForUnknownSentinel(t *testing.T) {
	// errors.Is with a sentinel that Error.Is doesn't recognize returns false.
	e := &Error{Status: 404, Name: "NotFoundError"}
	other := errors.New("unrelated sentinel")
	if errors.Is(e, other) {
		t.Errorf("errors.Is(404 err, unrelated) should be false")
	}
}
