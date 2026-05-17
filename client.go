package strapi

import (
	"net/http"
	"time"
)

const defaultUserAgent = "strapi-go/0.1 (+https://github.com/jorgejr568/strapi-go)"

// Client is the entry point to the Strapi API. Construct it with New and
// pass it to Collection[T] / SingleType[T] or the top-level helpers.
type Client struct {
	baseURL    string
	token      string
	userAgent  string
	apiVersion APIVersion
	httpClient *http.Client
}

// New constructs a Client. WithBaseURL is required; calling New without it
// panics.
func New(opts ...Option) *Client {
	c := &Client{
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.baseURL == "" {
		panic("strapi: WithBaseURL is required")
	}
	return c
}

// BaseURL returns the configured Strapi base URL (without trailing slash).
func (c *Client) BaseURL() string { return c.baseURL }

// HTTPClient returns the underlying *http.Client.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }
