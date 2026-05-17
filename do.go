package strapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// do builds and executes an HTTP request against the configured Strapi
// instance, decoding 2xx responses into dst and non-2xx responses into a
// typed *Error.
//
// Params:
//   - method:   "GET" | "POST" | "PUT" | "DELETE"
//   - path:     "/api/<endpoint>[/<documentId>]"
//   - rawQuery: query string without leading '?', or "" for none
//   - body:     any value to JSON-encode as the request body, or nil
//   - dst:      target for 2xx response decoding, or nil to discard
func (c *Client) do(ctx context.Context, method, path, rawQuery string, body, dst any) error {
	if c.apiVersion == APIVersionV4 {
		rawQuery = translateStatusToPublicationState(rawQuery)
	}
	url := c.baseURL + path
	if rawQuery != "" {
		if strings.Contains(url, "?") {
			url += "&" + rawQuery
		} else {
			url += "?" + rawQuery
		}
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("strapi: marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("strapi: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("strapi: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("strapi: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp.StatusCode, respBody)
	}

	if dst == nil || len(respBody) == 0 {
		return nil
	}
	if c.apiVersion == APIVersionV4 {
		normalized, err := normalizeV4ToV5(respBody)
		if err != nil {
			return fmt.Errorf("%w: v4 normalize: %v", ErrBadResponse, err)
		}
		respBody = normalized
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		return fmt.Errorf("%w: %v", ErrBadResponse, err)
	}
	return nil
}

func decodeError(status int, body []byte) error {
	var envelope struct {
		Error *Error `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		if envelope.Error.Status == 0 {
			envelope.Error.Status = status
		}
		return envelope.Error
	}
	return &Error{
		Status:  status,
		Name:    "UnknownError",
		Message: strings.TrimSpace(string(body)),
	}
}

// translateStatusToPublicationState rewrites a v5-shaped status= param to
// its v4 publicationState= equivalent in the given raw query string. It is
// a no-op when the query is empty or no status param is present.
//
// Mapping:
//
//	status=draft     → publicationState=preview  (v4: preview returns drafts AND published)
//	status=published → publicationState=live     (v4: live = published only)
//
// Note: v4 has no "drafts only" mode — preview is the closest equivalent.
func translateStatusToPublicationState(rawQuery string) string {
	if rawQuery == "" {
		return rawQuery
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	status := values.Get("status")
	if status == "" {
		return rawQuery
	}
	switch status {
	case "draft":
		values.Set("publicationState", "preview")
	case "published":
		values.Set("publicationState", "live")
	default:
		return rawQuery
	}
	values.Del("status")
	return values.Encode()
}
