package runtime

type SiteCreateCapability struct {
	Supported            bool     `json:"supported"`
	Endpoint             string   `json:"endpoint,omitempty"`
	CreateStrategy       string   `json:"create_strategy,omitempty"`
	DefaultEnabled       bool     `json:"default_enabled"`
	SemanticBackendTypes []string `json:"semantic_backend_types"`
	RequestBackendTypes  []string `json:"request_backend_types"`
	SemanticFlags        []string `json:"semantic_flags"`
	RequiredJSONFields   []string `json:"required_json_fields"`
	OptionalJSONFields   []string `json:"optional_json_fields"`
	Notes                []string `json:"notes"`
}

func SiteCreateCapabilities(ctx Context) SiteCreateCapability {
	caps := SiteCreateCapability{
		Supported:      ctx.SiteCreateSupported,
		Endpoint:       ctx.Endpoint,
		DefaultEnabled: false,
		Notes: []string{
			"site is disabled by default; use --enable to create enabled site",
			"response backend is supported only through --request JSON",
		},
	}
	if !ctx.SiteCreateSupported {
		return caps
	}

	caps.RequiredJSONFields = []string{"name", "server_names", "ports", "policy_group", "session_method"}
	caps.OptionalJSONFields = []string{"asset_group", "remark", "url_paths", "ssl_cert"}
	if ctx.VersionFamily == Family25_03 {
		caps.OptionalJSONFields = append(caps.OptionalJSONFields, "detector_ip_source_from", "ssl_gm_cert", "ports[].is_double_cert")
	}

	switch ctx.OperationMode {
	case ModeSoftwareReverseProxy, ModeHardwareReverseProxy, ModeHardwareRouterProxy:
		caps.CreateStrategy = "semantic"
		caps.SemanticBackendTypes = []string{"proxy", "redirect"}
		caps.RequestBackendTypes = []string{"proxy", "redirect", "response"}
		caps.SemanticFlags = []string{"--name", "--domain", "--port", "--ssl", "--cert-id", "--http2", "--sni", "--non-http", "--url-path", "--url-path-op", "--backend-type", "--upstream", "--load-balance", "--xff-action", "--redirect-url", "--redirect-code", "--policy-group", "--enable", "--request", "--check", "--explain", "--yes"}
	case ModeSoftwareClusterReverseProxy:
		caps.CreateStrategy = "semantic_limited"
		caps.SemanticBackendTypes = nil
		caps.RequestBackendTypes = nil
		caps.SemanticFlags = []string{"--name", "--domain", "--port", "--ssl", "--cert-id", "--http2", "--sni", "--url-path", "--url-path-op", "--policy-group", "--enable", "--request", "--check", "--explain", "--yes"}
		caps.Notes = append(caps.Notes, "cluster reverse proxy creation does not accept backend_config/upstream semantic flags")
	case ModeHardwareTransparentProxy, ModeHardwareTransparentBridging, ModeSoftwarePortMirroring, ModeHardwarePortMirroring, ModeHardwareTrafficDetection:
		caps.CreateStrategy = "request_only"
		caps.SemanticBackendTypes = nil
		caps.RequestBackendTypes = nil
		caps.SemanticFlags = []string{"--request", "--enable", "--check", "--explain", "--yes"}
		caps.Notes = append(caps.Notes, "this deployment mode requires --request JSON for site create")
	}
	return caps
}
