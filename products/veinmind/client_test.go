package veinmind

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientInjectsJSONHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/test", nil)
	client := NewClient(&Config{}, nil, false)

	client.injectHeaders(req, nil, false)

	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization header = %q, want empty", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("Accept = %q, want %q", got, "application/json")
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestClientAuthenticatesWithTokenLogin(t *testing.T) {
	dryRun = false

	const token = "12345678901234567890123456789012abcdefghijklmnopqrstuvwxyzABCDEF"
	apiSecret, apiKey, err := splitAPIToken(token)
	if err != nil {
		t.Fatalf("splitAPIToken() error = %v", err)
	}

	client := NewClient(&Config{
		URL:    "https://example.com",
		APIKey: token,
	}, nil, false)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/auth/token_login":
			var payload struct {
				Key     string `json:"key"`
				Sign    string `json:"sign"`
				NowTime int64  `json:"now_time"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode auth payload: %v", err)
			}
			if payload.Key != apiKey {
				t.Fatalf("auth key = %q, want %q", payload.Key, apiKey)
			}
			if want := generateTokenSign(apiKey, apiSecret, payload.NowTime); payload.Sign != want {
				t.Fatalf("auth sign = %q, want %q", payload.Sign, want)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"Set-Cookie":   []string{"veinmind=session-id; Path=/"},
				},
				Body:       io.NopCloser(strings.NewReader(`{"code":200}`)),
				Request:    r,
			}, nil
		case "/api/test":
			cookie, err := r.Cookie("veinmind")
			if err != nil {
				t.Fatalf("veinmind cookie missing: %v", err)
			}
			if cookie.Value != "session-id" {
				t.Fatalf("veinmind cookie = %q, want %q", cookie.Value, "session-id")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Request:    r,
			}, nil
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		return nil, nil
	})

	if err := client.Do(context.Background(), http.MethodGet, "/api/test", nil, nil, nil, nil); err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientReportsNonJSONResponseDetails(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(`<html><body>login</body></html>`)),
	}
	client := NewClient(&Config{}, nil, false)

	var result any
	err := client.handleResponse(resp, &result)
	if err == nil {
		t.Fatal("handleResponse() returned nil error")
	}

	message := err.Error()
	for _, want := range []string{"failed to parse JSON response", "text/html", "<html><body>login</body></html>"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}

func TestClientAuthRejectsNonJSONLoginResponse(t *testing.T) {
	dryRun = false

	const token = "12345678901234567890123456789012abcdefghijklmnopqrstuvwxyzABCDEF"
	client := NewClient(&Config{
		URL:    "https://example.com",
		APIKey: token,
	}, nil, false)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/auth/token_login":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(`<html><body>login</body></html>`)),
				Request:    r,
			}, nil
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		return nil, nil
	})

	err := client.Do(context.Background(), http.MethodGet, "/api/test", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("Do() returned nil error")
	}
	if !strings.Contains(err.Error(), "auth response is not JSON") {
		t.Fatalf("error %q does not contain %q", err.Error(), "auth response is not JSON")
	}
}

func TestClientAuthAllowsNonStandardCodeWhenSessionCookieExists(t *testing.T) {
	dryRun = false

	const token = "12345678901234567890123456789012abcdefghijklmnopqrstuvwxyzABCDEF"
	client := NewClient(&Config{
		URL:    "https://example.com",
		APIKey: token,
	}, nil, false)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/auth/token_login":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"Set-Cookie":   []string{"veinmind=session-id; Path=/"},
				},
				Body:       io.NopCloser(strings.NewReader(`{"code":401,"msg":"token invalid"}`)),
				Request:    r,
			}, nil
		case "/api/test":
			cookie, err := r.Cookie("veinmind")
			if err != nil {
				t.Fatalf("veinmind cookie missing: %v", err)
			}
			if cookie.Value != "session-id" {
				t.Fatalf("veinmind cookie = %q, want %q", cookie.Value, "session-id")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Request:    r,
			}, nil
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		return nil, nil
	})

	err := client.Do(context.Background(), http.MethodGet, "/api/test", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
}

func TestClientAuthRejectsMissingSessionCookie(t *testing.T) {
	dryRun = false

	const token = "12345678901234567890123456789012abcdefghijklmnopqrstuvwxyzABCDEF"
	client := NewClient(&Config{
		URL:    "https://example.com",
		APIKey: token,
	}, nil, false)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/auth/token_login":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"code":200}`)),
				Request:    r,
			}, nil
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		return nil, nil
	})

	err := client.Do(context.Background(), http.MethodGet, "/api/test", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("Do() returned nil error")
	}
	if !strings.Contains(err.Error(), "auth did not establish session") {
		t.Fatalf("error %q does not contain %q", err.Error(), "auth did not establish session")
	}
}
