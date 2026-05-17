// Package strapi is a Go SDK for the Strapi v5 REST API.
//
// It exposes a typed Client and two generic accessors — Collection[T] for
// plural collection-type endpoints (e.g. Pages, Articles) and SingleType[T]
// for single-record single-type endpoints (e.g. Homepage, Footer) — with a
// fluent query builder under github.com/jorgejr568/strapi-go/query, a
// rich-text block parser under .../blocks, and media helpers under .../media.
//
// MVP scope: read + write access to collection types (Find, List, Create,
// Update, Delete) and single types (Get, Update, Delete). File uploads are
// not yet supported.
package strapi
