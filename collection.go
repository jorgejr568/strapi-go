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
