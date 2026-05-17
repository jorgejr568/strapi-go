package query

import "strconv"

// Query accumulates Strapi REST query parameters. Build it via New(opts...)
// and serialize via Build, which returns a qs-compatible query string with
// no leading '?'.
type Query struct {
	enc encoder
}

// Option mutates a Query during construction.
type Option func(*Query)

// New builds a Query from the given options.
func New(opts ...Option) *Query {
	q := &Query{}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

// Build returns the encoded query string (no leading '?').
func (q *Query) Build() string {
	return q.enc.String()
}

// add is exposed to the package so filter.go and populate.go can write
// directly into the same ordered encoder.
func (q *Query) add(path []string, value string) {
	q.enc.add(path, value)
}

// Sort sorts results. Each entry is "field" or "field:asc"/"field:desc".
func Sort(fields ...string) Option {
	return func(q *Query) {
		for i, f := range fields {
			q.add([]string{"sort", strconv.Itoa(i)}, f)
		}
	}
}

// Paginate uses page-based pagination (Strapi default).
func Paginate(page, pageSize int) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "page"}, strconv.Itoa(page))
		q.add([]string{"pagination", "pageSize"}, strconv.Itoa(pageSize))
	}
}

// PaginateOffset uses offset-based pagination. Cannot be combined with
// Paginate — Strapi rejects mixed modes.
func PaginateOffset(start, limit int) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "start"}, strconv.Itoa(start))
		q.add([]string{"pagination", "limit"}, strconv.Itoa(limit))
	}
}

// WithCount controls whether pagination metadata includes the total count.
// Default is true on Strapi's side; set false to skip the COUNT query.
func WithCount(b bool) Option {
	return func(q *Query) {
		q.add([]string{"pagination", "withCount"}, strconv.FormatBool(b))
	}
}

// Fields selects which top-level fields are returned.
func Fields(names ...string) Option {
	return func(q *Query) {
		for i, n := range names {
			q.add([]string{"fields", strconv.Itoa(i)}, n)
		}
	}
}

// Locale restricts the query to a single locale. Use "all" for every locale.
func Locale(code string) Option {
	return func(q *Query) {
		q.add([]string{"locale"}, code)
	}
}

// PublicationStatus is the v5 replacement for v4's publicationState.
type PublicationStatus string

const (
	StatusDraft     PublicationStatus = "draft"
	StatusPublished PublicationStatus = "published"
)

// Status filters by draft/published state (v5).
func Status(s PublicationStatus) Option {
	return func(q *Query) {
		q.add([]string{"status"}, string(s))
	}
}
