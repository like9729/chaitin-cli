package safeline3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDoUnwrapsEnvelopeAndSendsToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("API-TOKEN"); got != "test-token" {
			t.Fatalf("API-TOKEN = %q, want test-token", got)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page query = %q, want 1", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"err": nil,
			"msg": "",
			"data": map[string]any{
				"ok": true,
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{URL: server.URL, APIToken: "test-token"}, false, false)
	query, err := parseQuery([]string{"page=1"})
	if err != nil {
		t.Fatalf("parseQuery() error = %v", err)
	}

	var result map[string]bool
	if err := client.Do(context.Background(), http.MethodGet, "/api/v3/test", query, nil, &result); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if !result["ok"] {
		t.Fatalf("result[ok] = false, want true")
	}
}

func TestParseQueryRejectsInvalidItem(t *testing.T) {
	if _, err := parseQuery([]string{"bad"}); err == nil {
		t.Fatal("parseQuery() error = nil, want error")
	}
}
