package strapi

import (
	"context"
	"net/http"

	"github.com/jorgejr568/strapi-go/query"
)

// Collection is a typed accessor for a Strapi collection-type endpoint.
// Construct it with NewCollection and reuse it across requests.
type Collection[T any] struct {
	client   *Client
	endpoint string // pluralApiId (e.g. "pages", "articles")
}

// NewCollection builds a Collection bound to the given endpoint
// (Strapi's pluralApiId, e.g. "pages").
func NewCollection[T any](c *Client, endpoint string) *Collection[T] {
	return &Collection[T]{client: c, endpoint: endpoint}
}

// Find fetches a single entry by its documentId (Strapi v5). Optional
// query options can populate relations, select fields, set locale, etc.
func (col *Collection[T]) Find(ctx context.Context, documentID string, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	if err := col.client.do(ctx, http.MethodGet, "/api/"+col.endpoint+"/"+documentID, q, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// List fetches a paginated list of entries. Use query options to filter,
// populate, sort, and paginate.
func (col *Collection[T]) List(ctx context.Context, opts ...query.Option) (*ListResponse[T], error) {
	q := query.New(opts...).Build()
	var env ListResponse[T]
	if err := col.client.do(ctx, http.MethodGet, "/api/"+col.endpoint, q, nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Find is a top-level convenience wrapper that builds a transient
// Collection[T] and calls Find.
func Find[T any](ctx context.Context, c *Client, endpoint, documentID string, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Find(ctx, documentID, opts...)
}

// List is a top-level convenience wrapper that builds a transient
// Collection[T] and calls List.
func List[T any](ctx context.Context, c *Client, endpoint string, opts ...query.Option) (*ListResponse[T], error) {
	return NewCollection[T](c, endpoint).List(ctx, opts...)
}

// Create inserts a new entry. attrs is the user-defined content for the
// entry (system fields are set by Strapi). The SDK wraps the payload as
// {"data": <attrs>} automatically. Optional query options apply to the
// returned representation (e.g. populate relations on the response).
func (col *Collection[T]) Create(ctx context.Context, attrs T, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := col.client.do(ctx, http.MethodPost, "/api/"+col.endpoint, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Update modifies an existing entry. attrs may be the full T or a partial
// payload (e.g. map[string]any{"title": "new"}) so callers can update
// individual fields without zeroing the rest. The SDK wraps the payload as
// {"data": <attrs>} automatically.
func (col *Collection[T]) Update(ctx context.Context, documentID string, attrs any, opts ...query.Option) (*Document[T], error) {
	q := query.New(opts...).Build()
	var env single[T]
	body := map[string]any{"data": attrs}
	if err := col.client.do(ctx, http.MethodPut, "/api/"+col.endpoint+"/"+documentID, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Delete removes an entry by documentId. The Strapi response body, if any,
// is discarded.
func (col *Collection[T]) Delete(ctx context.Context, documentID string) error {
	return col.client.do(ctx, http.MethodDelete, "/api/"+col.endpoint+"/"+documentID, "", nil, nil)
}

// Create is a top-level convenience wrapper.
func Create[T any](ctx context.Context, c *Client, endpoint string, attrs T, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Create(ctx, attrs, opts...)
}

// Update is a top-level convenience wrapper.
func Update[T any](ctx context.Context, c *Client, endpoint, documentID string, attrs any, opts ...query.Option) (*Document[T], error) {
	return NewCollection[T](c, endpoint).Update(ctx, documentID, attrs, opts...)
}

// Delete is a top-level convenience wrapper. Unlike Find/List/Create/Update,
// it is not generic — Strapi's delete endpoint returns no typed body, so
// there is no T to infer.
func Delete(ctx context.Context, c *Client, endpoint, documentID string) error {
	return (&Collection[any]{client: c, endpoint: endpoint}).Delete(ctx, documentID)
}
