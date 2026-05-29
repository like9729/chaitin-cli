package runtime

import "testing"

func TestSiteCreateCapabilitiesSoftwareReverseProxy(t *testing.T) {
	caps := SiteCreateCapabilities(Context{Version: "25.03.007_r7", VersionFamily: Family25_03, OperationMode: ModeSoftwareReverseProxy, Endpoint: "/api/SoftwareReverseProxyWebsiteAPI", SiteCreateSupported: true})
	if !caps.Supported {
		t.Fatalf("expected supported")
	}
	if caps.Endpoint != "/api/SoftwareReverseProxyWebsiteAPI" {
		t.Fatalf("bad endpoint %q", caps.Endpoint)
	}
	if !contains(caps.SemanticBackendTypes, "proxy") || !contains(caps.SemanticBackendTypes, "redirect") {
		t.Fatalf("missing semantic backend types: %+v", caps)
	}
	if !contains(caps.RequestBackendTypes, "response") {
		t.Fatalf("request mode should allow response")
	}
	if !contains(caps.OptionalJSONFields, "detector_ip_source_from") {
		t.Fatalf("25.03 should include detector_ip_source_from")
	}
}

func TestSiteCreateCapabilitiesCluster(t *testing.T) {
	caps := SiteCreateCapabilities(Context{Version: "23.01.014", VersionFamily: Family23_01, OperationMode: ModeSoftwareClusterReverseProxy, Endpoint: "/api/SoftwareClusterReverseProxyWebsiteAPI", SiteCreateSupported: true})
	if !caps.Supported {
		t.Fatalf("expected supported")
	}
	if contains(caps.SemanticFlags, "--upstream") {
		t.Fatalf("cluster semantic flags must not include upstream")
	}
	if len(caps.SemanticBackendTypes) != 0 {
		t.Fatalf("cluster should not advertise backend types: %+v", caps.SemanticBackendTypes)
	}
}

func TestSiteCreateCapabilitiesStrategies(t *testing.T) {
	tests := []struct {
		mode OperationMode
		want string
	}{
		{ModeSoftwareReverseProxy, "semantic"},
		{ModeHardwareReverseProxy, "semantic"},
		{ModeHardwareRouterProxy, "semantic"},
		{ModeSoftwareClusterReverseProxy, "semantic_limited"},
		{ModeHardwareTransparentProxy, "request_only"},
		{ModeHardwareTransparentBridging, "request_only"},
		{ModeSoftwarePortMirroring, "request_only"},
		{ModeHardwarePortMirroring, "request_only"},
		{ModeHardwareTrafficDetection, "request_only"},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			endpoint, ok := EndpointForMode(tt.mode)
			if !ok {
				t.Fatalf("missing endpoint for %q", tt.mode)
			}
			caps := SiteCreateCapabilities(Context{OperationMode: tt.mode, Endpoint: endpoint, SiteCreateSupported: true})
			if caps.CreateStrategy != tt.want {
				t.Fatalf("CreateStrategy = %q, want %q", caps.CreateStrategy, tt.want)
			}
		})
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
