package strapi

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := New(WithBaseURL("https://cms.example.com"))
	if c.BaseURL() != "https://cms.example.com" {
		t.Errorf("BaseURL = %q", c.BaseURL())
	}
	if c.HTTPClient() == nil {
		t.Error("HTTPClient should default to non-nil")
	}
	if c.HTTPClient().Timeout == 0 {
		t.Error("default timeout should be non-zero")
	}
}

func TestNewClientStripsTrailingSlash(t *testing.T) {
	c := New(WithBaseURL("https://cms.example.com/"))
	if c.BaseURL() != "https://cms.example.com" {
		t.Errorf("BaseURL = %q want trailing slash stripped", c.BaseURL())
	}
}

func TestNewClientWithToken(t *testing.T) {
	c := New(WithBaseURL("https://x"), WithToken("secret"))
	if c.token != "secret" {
		t.Errorf("token = %q", c.token)
	}
}

func TestNewClientWithCustomHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 5 * time.Second}
	c := New(WithBaseURL("https://x"), WithHTTPClient(hc))
	if c.HTTPClient() != hc {
		t.Error("WithHTTPClient should set the http.Client")
	}
}

func TestNewClientPanicsWithoutBaseURL(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when base URL missing")
		}
	}()
	_ = New()
}

func TestNewClientWithUserAgent(t *testing.T) {
	c := New(WithBaseURL("https://x"), WithUserAgent("my-app/1.0"))
	if c.userAgent != "my-app/1.0" {
		t.Errorf("userAgent = %q want %q", c.userAgent, "my-app/1.0")
	}
}

func TestNewClientDefaultsToV5(t *testing.T) {
	c := New(WithBaseURL("https://x"))
	if c.apiVersion != APIVersionV5 {
		t.Errorf("apiVersion = %v want APIVersionV5", c.apiVersion)
	}
}

func TestNewClientWithAPIVersionV4(t *testing.T) {
	c := New(WithBaseURL("https://x"), WithAPIVersion(APIVersionV4))
	if c.apiVersion != APIVersionV4 {
		t.Errorf("apiVersion = %v want APIVersionV4", c.apiVersion)
	}
}
