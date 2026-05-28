package site

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"

	safelineruntime "github.com/chaitin/chaitin-cli/products/safeline/runtime"
)

type createOptions struct {
	Name          string
	Domains       []string
	Ports         []int
	SSL           bool
	CertID        int
	HTTP2         bool
	SNI           bool
	NonHTTP       bool
	URLPath       string
	URLPathOp     string
	BackendType   string
	Upstreams     []string
	LoadBalance   string
	XFFAction     string
	RedirectURL   string
	RedirectCode  int
	PolicyGroupID int
	Enable        bool
	Request       json.RawMessage
	Check         bool
	Explain       bool
	Yes           bool
}

func buildCreatePayload(ctx safelineruntime.Context, opts createOptions) (map[string]any, error) {
	if len(opts.Request) > 0 {
		return normalizeRequestPayload(opts.Request, opts.Enable)
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	if len(opts.Domains) == 0 {
		return nil, fmt.Errorf("--domain is required and may be repeated; '*' is accepted")
	}
	if len(opts.Ports) == 0 {
		return nil, fmt.Errorf("--port is required and may be repeated")
	}
	if opts.URLPath == "" {
		opts.URLPath = "/"
	}
	if opts.URLPathOp == "" {
		opts.URLPathOp = "pre"
	}
	if opts.BackendType == "" {
		opts.BackendType = "proxy"
	}
	if opts.LoadBalance == "" {
		opts.LoadBalance = "Round Robin"
	}
	if opts.XFFAction == "" {
		opts.XFFAction = "append"
	}
	if opts.SSL && opts.CertID == 0 {
		return nil, fmt.Errorf("--cert-id is required when --ssl is set")
	}
	if opts.HTTP2 && !opts.SSL {
		return nil, fmt.Errorf("--http2 requires --ssl")
	}
	if opts.SNI && !opts.SSL {
		return nil, fmt.Errorf("--sni requires --ssl")
	}

	ports := make([]map[string]any, 0, len(opts.Ports))
	for _, port := range opts.Ports {
		if port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid --port %d", port)
		}
		ports = append(ports, map[string]any{"port": port, "ssl": opts.SSL, "http2": opts.HTTP2, "sni": opts.SNI, "non_http": opts.NonHTTP})
	}

	payload := map[string]any{
		"name":             opts.Name,
		"server_names":     opts.Domains,
		"ports":            ports,
		"url_paths":        []map[string]any{{"url_path": opts.URLPath, "op": opts.URLPathOp}},
		"session_method":   map[string]any{"type": "off"},
		"selected_tengine": map[string]any{"type": "all", "tengine_list": nil},
		"is_enabled":       opts.Enable,
	}
	if opts.PolicyGroupID == 0 {
		return nil, fmt.Errorf("--policy-group is required")
	}
	payload["policy_group"] = opts.PolicyGroupID
	if opts.SSL {
		payload["ssl_cert"] = opts.CertID
	}

	switch ctx.OperationMode {
	case safelineruntime.ModeSoftwareReverseProxy:
		backend, err := buildReverseProxyBackend(opts)
		if err != nil {
			return nil, err
		}
		payload["backend_config"] = backend
	case safelineruntime.ModeSoftwareClusterReverseProxy:
		if opts.BackendType != "" && opts.BackendType != "proxy" {
			return nil, fmt.Errorf("cluster reverse proxy does not support --backend-type")
		}
		if len(opts.Upstreams) > 0 || opts.RedirectURL != "" || (opts.RedirectCode != 0 && opts.RedirectCode != 302) {
			return nil, fmt.Errorf("cluster reverse proxy semantic create does not accept backend, upstream, or redirect flags")
		}
	default:
		return nil, fmt.Errorf("site create is unsupported for operation mode %q", ctx.OperationMode)
	}
	return payload, nil
}

func buildReverseProxyBackend(opts createOptions) (map[string]any, error) {
	switch opts.BackendType {
	case "proxy":
		if len(opts.Upstreams) == 0 {
			return nil, fmt.Errorf("--upstream is required for --backend-type proxy")
		}
		servers := make([]map[string]any, 0, len(opts.Upstreams))
		for _, u := range opts.Upstreams {
			server, err := parseBackendServer(u)
			if err != nil {
				return nil, err
			}
			servers = append(servers, server)
		}
		return map[string]any{"type": "proxy", "servers": servers, "load_balance_policy": opts.LoadBalance, "x_forwarded_for_action": opts.XFFAction}, nil
	case "redirect":
		if opts.RedirectURL == "" {
			return nil, fmt.Errorf("--redirect-url is required for --backend-type redirect")
		}
		code := opts.RedirectCode
		if code == 0 {
			code = 302
		}
		if code != 301 && code != 302 && code != 307 && code != 308 {
			return nil, fmt.Errorf("--redirect-code must be one of 301, 302, 307, 308")
		}
		return map[string]any{"type": "redirect", "redirect_url": opts.RedirectURL, "redirect_code": code}, nil
	default:
		return nil, fmt.Errorf("--backend-type must be proxy or redirect")
	}
}

func parseBackendServer(raw string) (map[string]any, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("--upstream must be a URL such as http://10.0.0.1:8080")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("--upstream protocol must be http or https")
	}
	host := u.Hostname()
	port := 80
	if u.Scheme == "https" {
		port = 443
	}
	if rawPort := u.Port(); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return nil, fmt.Errorf("invalid upstream port %q", rawPort)
		}
		port = parsed
	} else if _, _, err := net.SplitHostPort(u.Host); err != nil && u.Host != host {
		return nil, fmt.Errorf("invalid upstream host %q", u.Host)
	}
	return map[string]any{"protocol": u.Scheme, "host": host, "port": port, "weight": 1, "is_enabled": true}, nil
}

func normalizeRequestPayload(raw json.RawMessage, enable bool) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid --request JSON: %w", err)
	}
	if _, ok := payload["is_enabled"]; !ok {
		payload["is_enabled"] = false
	}
	if enable {
		payload["is_enabled"] = true
	}
	if _, ok := payload["name"]; !ok {
		return nil, fmt.Errorf("--request JSON must include name")
	}
	if _, ok := payload["server_names"]; !ok {
		return nil, fmt.Errorf("--request JSON must include server_names")
	}
	if _, ok := payload["ports"]; !ok {
		return nil, fmt.Errorf("--request JSON must include ports")
	}
	return payload, nil
}
