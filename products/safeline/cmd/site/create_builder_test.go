package site

import (
	"encoding/json"
	"testing"

	safelineruntime "github.com/chaitin/chaitin-cli/products/safeline/runtime"
)

func TestBuildSoftwareReverseProxyPayload(t *testing.T) {
	opts := createOptions{
		Name:          "app-a",
		Domains:       []string{"*"},
		Ports:         []int{443},
		SSL:           true,
		CertID:        12,
		HTTP2:         true,
		SNI:           true,
		URLPath:       "/",
		URLPathOp:     "pre",
		BackendType:   "proxy",
		Upstreams:     []string{"http://10.0.0.1:8080"},
		LoadBalance:   "Round Robin",
		XFFAction:     "append",
		PolicyGroupID: 3,
	}
	payload, err := buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareReverseProxy, VersionFamily: safelineruntime.Family25_03}, opts)
	if err != nil {
		t.Fatalf("buildCreatePayload: %v", err)
	}
	if payload["is_enabled"].(bool) {
		t.Fatalf("site must be disabled by default")
	}
	if got := payload["name"]; got != "app-a" {
		t.Fatalf("name %v", got)
	}
	if payload["session_method"].(map[string]any)["type"] != "off" {
		t.Fatalf("bad session_method %+v", payload["session_method"])
	}
	if payload["selected_tengine"].(map[string]any)["type"] != "all" {
		t.Fatalf("bad selected_tengine %+v", payload["selected_tengine"])
	}
	ports := payload["ports"].([]map[string]any)
	if ports[0]["port"] != 443 || ports[0]["ssl"] != true || ports[0]["http2"] != true || ports[0]["sni"] != true {
		t.Fatalf("bad ports %+v", ports)
	}
	if payload["ssl_cert"] != 12 {
		t.Fatalf("bad ssl_cert %+v", payload["ssl_cert"])
	}
	if payload["policy_group"] != 3 {
		t.Fatalf("bad policy_group %+v", payload["policy_group"])
	}
	if _, ok := payload["detector_ip_source_from"]; ok {
		t.Fatalf("semantic create must not set detector_ip_source_from: %+v", payload)
	}
	if _, ok := payload["detector_ip_source"]; ok {
		t.Fatalf("semantic create must let backend default detector_ip_source: %+v", payload)
	}
	backend := payload["backend_config"].(map[string]any)
	if backend["load_balance_policy"] != "Round Robin" || backend["x_forwarded_for_action"] != "append" {
		t.Fatalf("bad backend fields %+v", backend)
	}
	servers := backend["servers"].([]map[string]any)
	if servers[0]["protocol"] != "http" || servers[0]["host"] != "10.0.0.1" || servers[0]["port"] != 8080 || servers[0]["weight"] != 1 || servers[0]["is_enabled"] != true {
		t.Fatalf("bad backend server %+v", servers[0])
	}
}

func TestBuildRedirectPayload(t *testing.T) {
	opts := createOptions{Name: "redir", Domains: []string{"old.example.com"}, Ports: []int{80}, URLPath: "/", URLPathOp: "pre", BackendType: "redirect", RedirectURL: "https://new.example.com", RedirectCode: 301, PolicyGroupID: 1}
	payload, err := buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareReverseProxy, VersionFamily: safelineruntime.Family23_01}, opts)
	if err != nil {
		t.Fatalf("buildCreatePayload: %v", err)
	}
	backend := payload["backend_config"].(map[string]any)
	if backend["type"] != "redirect" || backend["redirect_url"] != "https://new.example.com" || backend["redirect_code"] != 301 {
		t.Fatalf("bad backend %+v", backend)
	}
	if _, ok := payload["detector_ip_source_from"]; ok {
		t.Fatalf("23.01 payload must not include detector_ip_source_from")
	}
}

func TestBuildClusterPayloadRejectsBackendFlags(t *testing.T) {
	opts := createOptions{Name: "cluster-a", Domains: []string{"api.example.com"}, Ports: []int{80}, URLPath: "/", URLPathOp: "pre", BackendType: "proxy", Upstreams: []string{"http://10.0.0.1"}, PolicyGroupID: 1}
	_, err := buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareClusterReverseProxy, VersionFamily: safelineruntime.Family23_01}, opts)
	if err == nil {
		t.Fatalf("expected cluster backend flag error")
	}
}

func TestBuildClusterPayloadAllowsDefaultRedirectCode(t *testing.T) {
	opts := createOptions{Name: "cluster-a", Domains: []string{"api.example.com"}, Ports: []int{80}, URLPath: "/", URLPathOp: "pre", BackendType: "proxy", RedirectCode: 302, PolicyGroupID: 1}
	_, err := buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareClusterReverseProxy, VersionFamily: safelineruntime.Family23_01}, opts)
	if err != nil {
		t.Fatalf("default redirect code must not make cluster create fail: %v", err)
	}
}

func TestValidateSSLRelationships(t *testing.T) {
	_, err := buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareReverseProxy}, createOptions{Name: "bad", Domains: []string{"a.example.com"}, Ports: []int{443}, SSL: true, URLPath: "/", URLPathOp: "pre", PolicyGroupID: 1})
	if err == nil {
		t.Fatalf("ssl without cert must fail")
	}
	_, err = buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareReverseProxy}, createOptions{Name: "bad", Domains: []string{"a.example.com"}, Ports: []int{80}, HTTP2: true, URLPath: "/", URLPathOp: "pre", PolicyGroupID: 1})
	if err == nil {
		t.Fatalf("http2 without ssl must fail")
	}
	_, err = buildCreatePayload(safelineruntime.Context{OperationMode: safelineruntime.ModeSoftwareReverseProxy}, createOptions{Name: "bad", Domains: []string{"a.example.com"}, Ports: []int{80}, SNI: true, URLPath: "/", URLPathOp: "pre", PolicyGroupID: 1})
	if err == nil {
		t.Fatalf("sni without ssl must fail")
	}
}

func TestNormalizeRequestJSONAppliesEnableOverride(t *testing.T) {
	raw := json.RawMessage(`{"name":"json-site","server_names":["a.example.com"],"ports":[{"port":80,"ssl":false,"http2":false}],"policy_group":{"id":1},"session_method":"ip_hash"}`)
	payload, err := normalizeRequestPayload(raw, true)
	if err != nil {
		t.Fatalf("normalizeRequestPayload: %v", err)
	}
	if payload["is_enabled"] != true {
		t.Fatalf("--enable must override is_enabled: %+v", payload)
	}
}
