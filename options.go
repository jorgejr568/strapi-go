package strapi

import "net/http"

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
