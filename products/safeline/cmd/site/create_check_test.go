package site

import "testing"

func TestLocalConflictChecks(t *testing.T) {
	payload := map[string]any{
		"server_names": []string{"a.example.com"},
		"ports":        []map[string]any{{"port": 80, "ssl": false}, {"port": 80, "ssl": true}},
		"url_paths":    []map[string]any{{"url_path": "/", "op": "pre"}},
	}
	warnings, errors := localCreateChecks(payload)
	if len(warnings) == 0 {
		t.Fatalf("expected warning for mixed ssl on same port")
	}
	if len(errors) != 0 {
		t.Fatalf("unexpected errors %+v", errors)
	}
}

func TestLocalConflictChecksAcceptsJSONPorts(t *testing.T) {
	payload := map[string]any{
		"server_names": []any{"a.example.com"},
		"ports":        []any{map[string]any{"port": float64(80), "ssl": false}},
		"url_paths":    []any{map[string]any{"url_path": "/", "op": "pre"}},
	}
	_, errors := localCreateChecks(payload)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors %+v", errors)
	}
}

func TestCheckResultShape(t *testing.T) {
	r := newCheckResult("/api/SoftwareReverseProxyWebsiteAPI", map[string]any{"name": "app"}, []string{"w"}, []string{})
	if !r.OK {
		t.Fatalf("expected ok")
	}
	if r.Operation != "site.create.check" {
		t.Fatalf("operation %q", r.Operation)
	}
	if r.Data.Endpoint != "/api/SoftwareReverseProxyWebsiteAPI" {
		t.Fatalf("endpoint %q", r.Data.Endpoint)
	}
}
