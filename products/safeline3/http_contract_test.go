package safeline3

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSiteCreateReverseProxyRequestContract(t *testing.T) {
	var createBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/node_group/list":
			writeEnvelope(t, w, map[string]any{"items": []map[string]any{{"id": 1, "mode": "ReverseProxy"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/protected-object/reverse-proxy":
			decodeBody(t, r, &createBody)
			writeEnvelope(t, w, createBody)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	out := executeSafeline3Command(t, server.URL,
		"site", "create", "reverse-proxy",
		"--name", "app",
		"--node-group", "1",
		"--domain", "app.example.com",
		"--listener", "10",
		"--upstream", "http://127.0.0.1:18080",
		"--yes",
	)

	if out == "" {
		t.Fatal("expected command output")
	}
	app := createBody["applications"].([]any)[0].(map[string]any)
	urlPath := app["url_paths"].([]any)[0].(map[string]any)
	if got := urlPath["path"]; got != "/" {
		t.Fatalf("url_paths[0].path = %#v, want /", got)
	}
	if _, exists := urlPath["url_path"]; exists {
		t.Fatalf("url_paths[0] should not contain legacy url_path key: %#v", urlPath)
	}
	action := app["reverse_proxy_action"].(map[string]any)
	backend := action["backend_config"].(map[string]any)
	lb := backend["load_balancing_config"].(map[string]any)
	if got := lb["method"]; got != "RoundRobin" {
		t.Fatalf("load_balancing_config.method = %#v, want RoundRobin", got)
	}
	if _, exists := lb["policy"]; exists {
		t.Fatalf("load_balancing_config should not contain legacy policy key: %#v", lb)
	}
}

func TestListenerListRequestContract(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/protected-object/listener" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		decodeBody(t, r, &body)
		writeEnvelope(t, w, map[string]any{"items": []any{}, "total": 0})
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL, "listener", "list", "--node-group", "1", "--port", "18082")

	assertCondition(t, body["node_group"], "=", float64(1))
	assertCondition(t, body["port"], "=", float64(18082))
}

func TestListenerCreateRouteProxyRequestContract(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/node_group/list":
			writeEnvelope(t, w, map[string]any{"items": []map[string]any{{"id": 2, "mode": "RouteProxy"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/protected-object/route-proxy/listener":
			decodeBody(t, r, &body)
			writeEnvelope(t, w, body)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL,
		"listener", "create", "route-proxy",
		"--name", "route-http",
		"--node-group", "2",
		"--ip", "10.0.0.0/24",
		"--port", "80",
		"--inbound-ip", "10.0.0.10",
		"--ntlm",
		"--transparent-server",
		"--protected-object", "7",
		"--yes",
	)

	if got := body["ntlm"]; got != true {
		t.Fatalf("ntlm = %#v, want true", got)
	}
	if got := body["transparent_server"]; got != true {
		t.Fatalf("transparent_server = %#v, want true", got)
	}
	if got := body["inbound_ips"]; !reflect.DeepEqual(got, []any{"10.0.0.10"}) {
		t.Fatalf("inbound_ips = %#v, want [10.0.0.10]", got)
	}
	if got := body["protected_objects"]; !reflect.DeepEqual(got, []any{float64(7)}) {
		t.Fatalf("protected_objects = %#v, want [7]", got)
	}
}

func TestListenerCreateTransparentProxyRequestContract(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/node_group/list":
			writeEnvelope(t, w, map[string]any{"items": []map[string]any{{"id": 3, "mode": "Transparent"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/protected-object/transparent-proxy/listener":
			decodeBody(t, r, &body)
			writeEnvelope(t, w, body)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL,
		"listener", "create", "transparent-proxy",
		"--name", "tp-http",
		"--node-group", "3",
		"--ip", "10.0.0.10",
		"--port", "80",
		"--virtual-wire-pair", "wire0",
		"--transparent-port",
		"--yes",
	)

	if got := body["virtual_wire_pair"]; got != "wire0" {
		t.Fatalf("virtual_wire_pair = %#v, want wire0", got)
	}
	if got := body["transparent_port"]; got != true {
		t.Fatalf("transparent_port = %#v, want true", got)
	}
}

func TestDefaultNodeGroupUpdateAndDeleteAreRejectedLocally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/node_group/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		writeEnvelope(t, w, map[string]any{"items": []map[string]any{{"id": 1, "mode": "ReverseProxy", "is_default": true}}})
	}))
	defer server.Close()

	if out := executeSafeline3CommandError(t, server.URL, "node-group", "update", "1", "--name", "x", "--yes"); !strings.Contains(out, "default node group 1 cannot be updated") {
		t.Fatalf("update error = %q", out)
	}
	if out := executeSafeline3CommandError(t, server.URL, "node-group", "delete", "1", "--yes"); !strings.Contains(out, "default node group 1 cannot be deleted") {
		t.Fatalf("delete error = %q", out)
	}
}

func TestSiteListUsesProtectedObjectQuery(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/protected-object" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		decodeBody(t, r, &body)
		writeEnvelope(t, w, map[string]any{
			"total": 2,
			"items": []map[string]any{
				{"id": 1, "type": "ReverseProxy", "name": "app"},
				{"id": 2, "type": "Mirror", "name": "mirror"},
			},
		})
	}))
	defer server.Close()

	out := executeSafeline3Command(t, server.URL, "site", "list", "--type", "reverse-proxy", "--name", "app")

	assertCondition(t, body["name"], "=", "app")
	var res struct {
		Total int              `json:"total"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("parse output: %v\n%s", err, out)
	}
	if res.Total != 1 || len(res.Items) != 1 || res.Items[0]["type"] != "ReverseProxy" {
		t.Fatalf("filtered output = %#v", res)
	}
}

func TestIPGroupAddIPUsesFullUpdateContract(t *testing.T) {
	var updateBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/detect/ip_group":
			writeEnvelope(t, w, map[string]any{
				"total": 1,
				"items": []map[string]any{{
					"id":       1,
					"name":     "group",
					"comment":  "keep",
					"original": []string{"203.0.113.1"},
				}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v3/detect/ip_group":
			decodeBody(t, r, &updateBody)
			writeEnvelope(t, w, updateBody)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL, "ip-group", "add-ip", "1", "--ip", "203.0.113.2", "--yes")

	if got := updateBody["name"]; got != "group" {
		t.Fatalf("name = %#v, want current name", got)
	}
	if _, exists := updateBody["operation"]; exists {
		t.Fatalf("update body should not contain patch operation: %#v", updateBody)
	}
	values := updateBody["original"].([]any)
	if len(values) != 2 || values[0] != "203.0.113.1" || values[1] != "203.0.113.2" {
		t.Fatalf("original = %#v, want merged IP list", values)
	}
}

func TestPolicyRuleListRequestContract(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/detect/PolicyRule/filter" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		decodeBody(t, r, &body)
		writeEnvelope(t, w, map[string]any{"data": []any{}, "total": 0})
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL, "policy-rule", "list", "--name", "rule", "--enabled", "disabled", "--global=false", "--app-id", "7")

	assertValueObject(t, body["is_global"], false)
	assertValueObject(t, body["app_ids"], []any{float64(7)})
	assertCondition(t, body["name"], "=", "rule")
	assertCondition(t, body["status"], "=", "disabled")
}

func TestACLTemplateCreateRequestContract(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/acl/acl-template" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		decodeBody(t, r, &body)
		writeEnvelope(t, w, map[string]any{"id": 1})
	}))
	defer server.Close()

	_ = executeSafeline3Command(t, server.URL,
		"acl", "template", "create",
		"--name", "acl",
		"--mode", "forbidden",
		"--node-group", "1",
		"--scope", "all",
		"--target-type", "cidr",
		"--period", "60",
		"--limit", "10",
		"--action-type", "deny",
		"--enabled=false",
		"--yes",
	)

	if got := body["mode"]; got != "forbidden" {
		t.Fatalf("mode = %#v, want forbidden", got)
	}
	if got := body["is_enabled"]; got != false {
		t.Fatalf("is_enabled = %#v, want false", got)
	}
	match := body["match_method"].(map[string]any)
	if got := match["scope"]; got != "All" {
		t.Fatalf("scope = %#v, want All", got)
	}
	if got := match["target_type"]; got != "CIDR" {
		t.Fatalf("target_type = %#v, want CIDR", got)
	}
	action := body["action"].(map[string]any)
	forbidden := action["forbidden"].(map[string]any)
	if got := forbidden["status_code"]; got != float64(403) {
		t.Fatalf("forbidden.status_code = %#v, want 403", got)
	}
}

func executeSafeline3Command(t *testing.T, url string, args ...string) string {
	t.Helper()
	t.Cleanup(func() {
		runtimeCfg = Config{}
		dryRun = false
	})
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	fullArgs := append([]string{"--url", url, "--api-token", "test-token", "--output", "json"}, args...)
	cmd.SetArgs(fullArgs)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(%v) error = %v\n%s", args, err, out.String())
	}
	return out.String()
}

func executeSafeline3CommandError(t *testing.T, url string, args ...string) string {
	t.Helper()
	t.Cleanup(func() {
		runtimeCfg = Config{}
		dryRun = false
	})
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	fullArgs := append([]string{"--url", url, "--api-token", "test-token", "--output", "json"}, args...)
	cmd.SetArgs(fullArgs)
	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute(%v) expected error\n%s", args, out.String())
	}
	return out.String()
}

func assertValueObject(t *testing.T, value any, want any) {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value object = %#v, want object", value)
	}
	got := obj["value"]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("value = %#v, want %#v", got, want)
	}
}

func decodeBody(t *testing.T, r *http.Request, out any) {
	t.Helper()
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"err": nil, "msg": "", "data": data}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func assertCondition(t *testing.T, value any, operator string, want any) {
	t.Helper()
	cond, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("condition = %#v, want object", value)
	}
	if got := cond["operator"]; got != operator {
		t.Fatalf("operator = %#v, want %q", got, operator)
	}
	values, ok := cond["value"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("condition values = %#v, want single value", cond["value"])
	}
	if values[0] != want {
		t.Fatalf("condition value = %#v, want %#v", values[0], want)
	}
}
