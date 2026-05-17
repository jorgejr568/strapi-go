// Package media models Strapi media-library file entries and provides a
// helper to resolve relative URLs against a Strapi base URL.
package media

import (
	"strings"
	"time"
)

// File is a Strapi upload entry, returned either standalone via
// /api/upload or inlined as a populated relation.
type File struct {
	ID              int       `json:"id"`
	DocumentID      string    `json:"documentId,omitempty"`
	Name            string    `json:"name"`
	AlternativeText string    `json:"alternativeText,omitempty"`
	Caption         string    `json:"caption,omitempty"`
	Width           int       `json:"width,omitempty"`
	Height          int       `json:"height,omitempty"`
	Hash            string    `json:"hash"`
	Ext             string    `json:"ext"`
	MIME            string    `json:"mime"`
	Size            float64   `json:"size"`
	URL             string    `json:"url"`
	PreviewURL      string    `json:"previewUrl,omitempty"`
	Provider        string    `json:"provider"`
	Formats         *Formats  `json:"formats,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Formats holds the set of resized variants Strapi may have generated.
// Each variant is optional — Strapi only emits a format when the source
// image is larger than the breakpoint.
type Formats struct {
	Thumbnail *Format `json:"thumbnail,omitempty"`
	Small     *Format `json:"small,omitempty"`
	Medium    *Format `json:"medium,omitempty"`
	Large     *Format `json:"large,omitempty"`
}

// Format describes a single resized image variant.
type Format struct {
	Name   string  `json:"name"`
	Hash   string  `json:"hash"`
	Ext    string  `json:"ext"`
	MIME   string  `json:"mime"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Size   float64 `json:"size"`
	URL    string  `json:"url"`
}

// ResolveURL joins a relative Strapi media URL against the base URL.
// Absolute URLs (http://, https://) and data URIs are returned unchanged.
// An empty url returns an empty string.
func ResolveURL(baseURL, url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "data:") {
		return url
	}
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	return baseURL + url
}
