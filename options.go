package strapi_go

import "net/http"

type clientOption struct {
	httpClient  *http.Client
	strapiUrl   *string
	strapiToken *string
}

func WithHttpClient(client *http.Client) *clientOption {
	return &clientOption{
		httpClient: client,
	}
}

func WithStrapiUrl(url string) *clientOption {
	return &clientOption{
		strapiUrl: &url,
	}
}

func WithStrapiToken(token string) *clientOption {
	return &clientOption{
		strapiToken: &token,
	}
}

func handleClientOptions(options []*clientOption) *clientOption {
	finalOption := &clientOption{}
	for _, option := range options {
		if option.httpClient != nil {
			finalOption.httpClient = option.httpClient
		}

		if option.strapiUrl != nil {
			finalOption.strapiUrl = option.strapiUrl
		}

		if option.strapiUrl != nil {
			finalOption.strapiUrl = option.strapiUrl
		}
	}

	return finalOption
}

type filterOption struct {
	operation string
	column    string
}

func withFilterOption(operation string) func(column string) *filterOption {
	return func(column string) *filterOption {
		return &filterOption{
			operation: operation,
			column:    column,
		}
	}
}

var (
	WhereEqual            = withFilterOption("$eq")
	WhereEqualInsensitive = withFilterOption("$eqi")
	WhereNotEqual         = withFilterOption("$ne")
)

type selectOption struct {
}
