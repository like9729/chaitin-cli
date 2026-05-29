package runtime

import "testing"

func TestNormalizeOperationMode(t *testing.T) {
	tests := []struct {
		in   string
		want OperationMode
	}{
		{"Software Reverse Proxy", ModeSoftwareReverseProxy},
		{"software-reverse-proxy", ModeSoftwareReverseProxy},
		{"software_reverse_proxy", ModeSoftwareReverseProxy},
		{"Software Cluster Reverse Proxy", ModeSoftwareClusterReverseProxy},
	}

	for _, tt := range tests {
		got, err := NormalizeOperationMode(tt.in)
		if err != nil {
			t.Fatalf("NormalizeOperationMode(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("got %q want %q", got, tt.want)
		}
	}
}

func TestVersionFamily(t *testing.T) {
	tests := []struct {
		in   string
		want VersionFamily
	}{
		{"23.01.014", Family23_01},
		{"25.03.007_r7", Family25_03},
		{"25.03.009", Family25_03},
		{"", FamilyUnknown},
		{"26.01.001", FamilyUnknown},
	}

	for _, tt := range tests {
		if got := ClassifyVersion(tt.in); got != tt.want {
			t.Fatalf("%q got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestEndpointForMode(t *testing.T) {
	path, ok := EndpointForMode(ModeSoftwareReverseProxy)
	if !ok || path != "/api/SoftwareReverseProxyWebsiteAPI" {
		t.Fatalf("unexpected software reverse endpoint %q %v", path, ok)
	}

	path, ok = EndpointForMode(ModeSoftwareClusterReverseProxy)
	if !ok || path != "/api/SoftwareClusterReverseProxyWebsiteAPI" {
		t.Fatalf("unexpected cluster endpoint %q %v", path, ok)
	}

}

func TestEndpointForModeSupportsAllSiteCreateModes(t *testing.T) {
	tests := []struct {
		mode OperationMode
		want string
	}{
		{ModeSoftwareReverseProxy, "/api/SoftwareReverseProxyWebsiteAPI"},
		{ModeSoftwareClusterReverseProxy, "/api/SoftwareClusterReverseProxyWebsiteAPI"},
		{ModeHardwareReverseProxy, "/api/HardwareReverseProxyWebsiteAPI"},
		{ModeHardwareTransparentProxy, "/api/HardwareTransparentProxyWebsiteAPI"},
		{ModeHardwareTransparentBridging, "/api/HardwareTransparentBridgingWebsiteAPI"},
		{ModeSoftwarePortMirroring, "/api/SoftwarePortMirroringWebsiteAPI"},
		{ModeHardwarePortMirroring, "/api/HardwareTrafficDetectionWebsiteAPI"},
		{ModeHardwareTrafficDetection, "/api/HardwareTrafficDetectionWebsiteAPI"},
		{ModeHardwareRouterProxy, "/api/HardwareReverseProxyWebsiteAPI"},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got, ok := EndpointForMode(tt.mode)
			if !ok {
				t.Fatalf("EndpointForMode(%q) not supported", tt.mode)
			}
			if got != tt.want {
				t.Fatalf("EndpointForMode(%q) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}
