package strapi

import (
	"context"
	"net/http"

	"github.com/jorgejr568/strapi-go/query"
)

// SingleType is a typed accessor for a Strapi single-type endpoint. A single
// type has exactly one record at /api/:singularApiId — there is no
// documentId in the path and no List operation.
type SingleType[T any] struct {
	client   *Client
	endpoint string // singularApiId (e.g. "homepage")
}

// NewSingleType builds a SingleType bound to the given endpoint
// (Strapi's singularApiId, e.g. "homepage").
func NewSingleType[T any](c *Client, endpoint string) *SingleType[T] {
	return &SingleType[T]{client: c, endpoint: endpoint}
}

// Get fetches the single-type record. Query options can populate relations,
// set locale, or select draft/published status.
func (s *SingleType[T]) Get(ctx context.Context, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	if err := s.client.do(ctx, http.MethodGet, "/api/"+s.endpoint, q, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Update modifies the single-type record. attrs may be the full T or a
// partial payload (e.g. map[string]any). The SDK wraps it as {"data": attrs}.
func (s *SingleType[T]) Update(ctx context.Context, attrs any, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := s.client.do(ctx, http.MethodPut, "/api/"+s.endpoint, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Delete clears the single-type record. Strapi semantics vary by version —
// some treat this as deleting the content row; others as resetting to
// empty. Either way, the response body is discarded.
func (s *SingleType[T]) Delete(ctx context.Context) error {
	return s.client.do(ctx, http.MethodDelete, "/api/"+s.endpoint, "", nil, nil)
}

// GetSingle is a top-level convenience wrapper.
func GetSingle[T any](ctx context.Context, c *Client, endpoint string, opts ...query.Option) (*Document[T], error) {
	return NewSingleType[T](c, endpoint).Get(ctx, opts...)
}

// UpdateSingle is a top-level convenience wrapper.
func UpdateSingle[T any](ctx context.Context, c *Client, endpoint string, attrs any, opts ...query.Option) (*Document[T], error) {
	return NewSingleType[T](c, endpoint).Update(ctx, attrs, opts...)
}

// DeleteSingle is a top-level convenience wrapper.
func DeleteSingle(ctx context.Context, c *Client, endpoint string) error {
	return (&SingleType[any]{client: c, endpoint: endpoint}).Delete(ctx)
}
