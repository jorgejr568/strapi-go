// Package strapi is a Go SDK for the Strapi REST API. v5 is the default;
// v4 instances are supported via WithAPIVersion(APIVersionV4).
//
// It exposes a typed Client and two generic accessors — Collection[T] for
// plural collection-type endpoints (e.g. Pages, Articles) and SingleType[T]
// for single-record single-type endpoints (e.g. Homepage, Footer) — with a
// fluent query builder under github.com/jorgejr568/strapi-go/query, a
// rich-text block parser under .../blocks, and media helpers under .../media.
//
// In v4 mode the SDK transparently translates the v4 response envelope
// ({id, attributes: {...}} entries, {data: ...} relation wrappers) into
// the v5 flat shape that the typed accessors decode. The query builder is
// version-agnostic; the only call-site difference is that v4 entries have
// no DocumentID, so callers use Document.PathID() (or strconv.Itoa(doc.ID))
// to address records in subsequent operations.
//
// MVP scope: read + write access to collection types (Find, List, Create,
// Update, Delete) and single types (Get, Update, Delete). File uploads are
// not yet supported.
package strapi
