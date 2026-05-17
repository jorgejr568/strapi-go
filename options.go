package strapi

import "net/http"

// APIVersion selects the Strapi REST response format the SDK expects.
// V5 is the default; V4 enables transparent shape translation so the same
// typed accessors and query builder work against a Strapi v4 instance.
type APIVersion int

const (
	// APIVersionV5 is the default — Strapi v5 flat-entry response shape.
	APIVersionV5 APIVersion = iota
	// APIVersionV4 enables v4 response normalization (attributes lifting +
	// relation-envelope unwrapping) and the v5→v4 query-param translation
	// for `status`.
	APIVersionV4
)

// Option configures a Client during New.
type Option func(*Client)

// WithBaseURL sets the Strapi instance URL (e.g. "https://cms.example.com").
// Trailing slashes are stripped. Required.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		for len(url) > 0 && url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}
		c.baseURL = url
	}
}

// WithToken sets the Strapi API token sent as `Authorization: Bearer <token>`.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.httpClient = h
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithAPIVersion selects the Strapi REST format. Default is APIVersionV5.
// Use APIVersionV4 against a Strapi v4 instance.
func WithAPIVersion(v APIVersion) Option {
	return func(c *Client) {
		c.apiVersion = v
	}
}
