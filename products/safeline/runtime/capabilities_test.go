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

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
