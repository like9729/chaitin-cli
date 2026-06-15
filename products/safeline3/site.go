package safeline3

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var allSiteTypes = []string{"reverse-proxy", "route-proxy", "transparent-proxy", "transparent", "mirror", "sdk"}

type siteCreateOptions struct {
	writeOptions
	Name                     string
	NodeGroup                uint64
	Domains                  []string
	Enabled                  bool
	Comment                  string
	AppName                  string
	URLPaths                 []string
	URLPathOp                string
	State                    string
	PolicyGroup              uint64
	ApplicationFile          string
	DetectorConfigFile       string
	SessionFile              string
	AccessLogFile            string
	PayloadFile              string
	Listeners                []string
	CertID                   uint64
	ProxyDetectionConfigFile string
	CustomNginxConfigFile    string
	RemoteListeners          []string
	RemoteListenerFile       string
	BackendType              string
	Upstreams                []string
	LoadBalance              string
	BackendHTTP2             bool
	BackendNTLM              bool
	RedirectURL              string
	RedirectCode             int
}

func newSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Manage protected objects",
		Long:  "Manage SafeLine-3 protected objects. Object type must be compatible with the selected node group mode.",
	}
	cmd.AddCommand(newSiteCapabilitiesCommand())
	cmd.AddCommand(newSiteListCommand())
	cmd.AddCommand(newSiteGetCommand())
	cmd.AddCommand(newSiteCreateCommand())
	cmd.AddCommand(newSiteUpdateCommand())
	cmd.AddCommand(newSiteDeleteCommand())
	cmd.AddCommand(newSiteToggleCommand("enable", true))
	cmd.AddCommand(newSiteToggleCommand("disable", false))
	return cmd
}

func newSiteCapabilitiesCommand() *cobra.Command {
	var typ string
	var nodeGroup uint64
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "Show protected object creation capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			caps := make([]map[string]any, 0, len(allSiteTypes))
			mode := ""
			if nodeGroup != 0 {
				group, err := fetchNodeGroup(cmd, nodeGroup)
				if err != nil {
					return err
				}
				mode = stringField(group, "mode")
			}
			types := allSiteTypes
			if typ != "" {
				normalized, err := normalizeType(typ)
				if err != nil {
					return err
				}
				types = []string{normalized}
			}
			for _, t := range types {
				required := []string{"--name", "--node-group", "--domain", "--yes"}
				if isProxyObjectType(t) {
					required = append(required, "--listener")
				} else {
					required = append(required, "--remote-listener")
				}
				if t == "reverse-proxy" {
					required = append(required, "--upstream when --backend-type proxy")
				}
				supported := true
				if mode != "" {
					supported = isTypeSupportedByMode(t, mode)
				}
				caps = append(caps, map[string]any{
					"type":                     t,
					"supported":                supported,
					"create_strategy":          createStrategyForType(t),
					"required_flags":           required,
					"condition_required_flags": []string{"--cert-id when selected listener has TLS enabled"},
					"file_flags":               []string{"--payload-file", "--application-file", "--detector-config-file", "--session-file", "--access-log-file"},
					"notes":                    []string{"node group mode compatibility is checked before sending API requests"},
				})
			}
			return getRenderer(cmd).Render(map[string]any{"node_group": nodeGroup, "mode": modeCLI(mode), "capabilities": caps})
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type")
	cmd.Flags().Uint64Var(&nodeGroup, "node-group", 0, "Node group ID")
	return cmd
}

func newSiteListCommand() *cobra.Command {
	var typ, name, domain string
	var nodeGroup uint64
	var enabled string
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List protected objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var apiType string
			if typ != "" {
				normalized, err := normalizeType(typ)
				if err != nil {
					return err
				}
				apiType, err = apiProtectType(normalized)
				if err != nil {
					return err
				}
			}
			body := map[string]any{"offset": (page - 1) * pageSize, "count": pageSize}
			if name != "" {
				body["name"] = equalCondition(name)
			}
			if domain != "" {
				body["domain_names"] = equalCondition(domain)
			}
			if nodeGroup != 0 {
				body["node_group"] = equalCondition(nodeGroup)
			}
			if enabled != "" {
				parsed, err := strconv.ParseBool(enabled)
				if err != nil {
					return fmt.Errorf("--enabled must be true or false")
				}
				body["is_enabled"] = equalCondition(parsed)
			}
			var res struct {
				Total int              `json:"total"`
				Items []map[string]any `json:"items"`
			}
			if err := getClient(cmd).Do(context.Background(), http.MethodPost, "/api/v3/protected-object", nil, body, &res); err != nil {
				return err
			}
			if apiType != "" {
				res.Items = filterMaps(res.Items, func(item map[string]any) bool {
					return stringField(item, "type") == apiType
				})
				res.Total = len(res.Items)
			}
			if res.Items == nil {
				res.Items = []map[string]any{}
			}
			return getRenderer(cmd).Render(res)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name")
	cmd.Flags().StringVar(&domain, "domain", "", "Filter by domain")
	cmd.Flags().Uint64Var(&nodeGroup, "node-group", 0, "Filter by node group")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Filter by enabled state true|false")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newSiteGetCommand() *cobra.Command {
	var typ string
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a protected object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if typ != "" {
				value, err := fetchProtectedObject(cmd, typ, id)
				if err != nil {
					return err
				}
				return getRenderer(cmd).Render(value)
			}
			var lastErr error
			for _, t := range allSiteTypes {
				value, err := fetchProtectedObject(cmd, t, id)
				if err == nil {
					return getRenderer(cmd).Render(value)
				}
				lastErr = err
			}
			return fmt.Errorf("protected object %d not found: %v", id, lastErr)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type")
	return cmd
}

func newSiteCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a protected object",
		Long: `Create a SafeLine-3 protected object.

Use one subcommand per object type. The CLI checks --node-group mode before
sending the request and rejects incompatible object types locally.

Supported types:
  reverse-proxy, route-proxy, transparent-proxy, transparent, mirror, sdk`,
	}
	for _, typ := range allSiteTypes {
		t := typ
		opts := siteCreateOptions{AppName: "default", URLPathOp: "prefix", State: "detect", BackendType: "proxy", LoadBalance: "round-robin", RedirectCode: 302}
		sub := &cobra.Command{
			Use:   t,
			Short: "Create a " + t + " protected object",
			Long:  siteCreateLong(t),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runSiteCreate(cmd, t, opts)
			},
		}
		addSiteCreateFlags(sub, &opts, t)
		cmd.AddCommand(sub)
	}
	return cmd
}

func newSiteUpdateCommand() *cobra.Command {
	var opts siteCreateOptions
	opts.AppName = "default"
	opts.URLPathOp = "prefix"
	opts.State = "detect"
	opts.BackendType = "proxy"
	opts.LoadBalance = "round-robin"
	opts.RedirectCode = 302
	cmd := &cobra.Command{
		Use:   "update <type> <id>",
		Short: "Update a protected object",
		Long: `Update a protected object.

API:
  PUT /api/v3/protected-object/<type>

Required:
  <type> reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk
  <id> positive integer
  --yes

Behavior:
  Without --payload-file, the CLI reads the current object, merges the flags
  you provided, and sends a full update payload. With --payload-file, the file
  is used as the complete request body.

Examples:
  chaitin-cli safeline-3 site update reverse-proxy 12 --domain app.example.com --yes
  chaitin-cli safeline-3 site update transparent 30 --remote-listener 10.0.0.11:80 --check`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			typ, err := normalizeType(args[0])
			if err != nil {
				return err
			}
			id, err := parseID(args[1])
			if err != nil {
				return err
			}
			var body any
			if opts.PayloadFile != "" {
				body, err = readJSONFile(opts.PayloadFile)
				if err != nil {
					return err
				}
			} else {
				current, err := fetchProtectedObject(cmd, typ, id)
				if err != nil {
					return err
				}
				bodyMap, ok := current.(map[string]any)
				if !ok {
					return fmt.Errorf("unexpected protected object response")
				}
				bodyMap["id"] = id
				patch, err := buildSitePayload(cmd, typ, opts)
				if err != nil {
					return err
				}
				for k, v := range patch {
					bodyMap[k] = v
				}
				body = bodyMap
			}
			path, _ := pathForType(typ)
			return doWrite(cmd, opts.writeOptions, "site.update."+typ, http.MethodPut, path, nil, body, []string{"update reads current object and sends a full payload"})
		},
	}
	addSiteCreateFlags(cmd, &opts, "")
	return cmd
}

func newSiteDeleteCommand() *cobra.Command {
	var typ string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "delete <id...>",
		Short: "Delete protected objects",
		Long: `Delete protected objects.

API:
  DELETE /api/v3/protected-object/delete
  DELETE /api/v3/protected-object/<type> when --type is provided

Required:
  <id...> one or more positive integer IDs
  --yes

Optional:
  --type reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk

Examples:
  chaitin-cli safeline-3 site delete 12 13 --check
  chaitin-cli safeline-3 site delete 12 --type reverse-proxy --yes`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDs(args)
			if err != nil {
				return err
			}
			if typ != "" {
				t, err := normalizeType(typ)
				if err != nil {
					return err
				}
				for _, id := range ids {
					path, _ := pathForType(t)
					body := map[string]any{"id": id}
					if err := doWrite(cmd, opts, "site.delete."+t, http.MethodDelete, path, nil, body, nil); err != nil {
						return err
					}
				}
				return nil
			}
			return doWrite(cmd, opts, "site.delete", http.MethodDelete, "/api/v3/protected-object/delete", nil, map[string]any{"ids": ids}, nil)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newSiteToggleCommand(name string, enabled bool) *cobra.Command {
	var typ string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   name + " <id...>",
		Short: name + " protected objects",
		Long: fmt.Sprintf(`%s protected objects.

API:
  PUT /api/v3/protected-object/toggle

Required:
  <id...> one or more positive integer IDs
  --yes

Optional:
  --type is accepted for command clarity, but the current API toggle payload
  is ID based.

Examples:
  chaitin-cli safeline-3 site %s 12 --check
  chaitin-cli safeline-3 site %s 12 13 --yes`, strings.Title(name), name, name),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = typ
			ids, err := parseIDs(args)
			if err != nil {
				return err
			}
			body := map[string]any{"ids": ids, "enable": enabled}
			return doWrite(cmd, opts, "site."+name, httpMethodForEnabled(enabled), "/api/v3/protected-object/toggle", nil, body, nil)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type (accepted for help compatibility)")
	addWriteFlags(cmd, &opts)
	return cmd
}

func siteCreateLong(typ string) string {
	base := fmt.Sprintf(`Create a %s protected object.

API:
  POST /api/v3/protected-object/%s

Required:
  --name string
  --node-group ID
  --domain string        Repeatable or comma separated.
  --yes

Common Optional:
  --enabled
  --comment string
  --app-name string      Default: default.
  --url-path string      Repeatable or comma separated; default /.
  --url-path-op prefix|exact|regex
  --state detect|dry-run|bypass|forbidden|not-apply|redirect|response|cache
  --policy-group ID

File Fallback:
  --payload-file JSON    Full API body; do not combine with semantic fields.
  --application-file JSON object/array
  --detector-config-file JSON
  --session-file JSON
  --access-log-file JSON

Behavior:
  The CLI queries --node-group, checks mode compatibility, then sends the
  generated payload. Write operations require --yes; use --check to print the
  generated request without writing.`, typ, typ)

	if isProxyObjectType(typ) {
		base += `

Proxy Required:
  --listener ID          Repeatable or comma separated.

Proxy Optional:
  --cert-id ID           Required when selected listener uses TLS.
  --proxy-detection-config-file JSON
  --custom-nginx-config-file JSON`
	} else {
		base += `

Non-Proxy Required:
  --remote-listener IP:PORT or IP/CIDR:PORT

Non-Proxy Optional:
  --remote-listener-file JSON`
	}

	if typ == "reverse-proxy" {
		base += `

Reverse Proxy Backend:
  --backend-type proxy|redirect   Default: proxy.
  --upstream http://HOST:PORT[,weight=N]
      Required when --backend-type proxy. Repeatable.
  --redirect-url URL
      Required when --backend-type redirect.
  --redirect-code 301|302|307|308   Default: 302.

Examples:
  chaitin-cli safeline-3 site create reverse-proxy --name app --node-group 1 --domain app.example.com --listener 10 --upstream http://10.0.0.1:8080 --yes
  chaitin-cli safeline-3 site create reverse-proxy --name redirect-app --node-group 1 --domain old.example.com --listener 10 --backend-type redirect --redirect-url https://new.example.com --yes`
	} else if isProxyObjectType(typ) {
		base += fmt.Sprintf(`

Examples:
  chaitin-cli safeline-3 site create %s --name app --node-group 1 --domain app.example.com --listener 20 --yes
  chaitin-cli safeline-3 site create %s --payload-file object.json --check`, typ, typ)
	} else {
		base += fmt.Sprintf(`

Examples:
  chaitin-cli safeline-3 site create %s --name app --node-group 2 --domain app.internal --remote-listener 10.0.0.10:80 --yes
  chaitin-cli safeline-3 site create %s --payload-file object.json --check`, typ, typ)
	}
	return base
}

func addSiteCreateFlags(cmd *cobra.Command, opts *siteCreateOptions, typ string) {
	cmd.Flags().StringVar(&opts.Name, "name", "", "Protected object name")
	cmd.Flags().Uint64Var(&opts.NodeGroup, "node-group", 0, "Node group ID")
	cmd.Flags().StringArrayVar(&opts.Domains, "domain", nil, "Domain name; repeatable or comma separated")
	cmd.Flags().BoolVar(&opts.Enabled, "enabled", false, "Create object enabled")
	cmd.Flags().StringVar(&opts.Comment, "comment", "", "Comment")
	cmd.Flags().StringVar(&opts.AppName, "app-name", "default", "Default application name")
	cmd.Flags().StringArrayVar(&opts.URLPaths, "url-path", nil, "URL path; repeatable or comma separated")
	cmd.Flags().StringVar(&opts.URLPathOp, "url-path-op", "prefix", "URL path op (prefix|exact|regex)")
	cmd.Flags().StringVar(&opts.State, "state", "detect", "Protected state")
	cmd.Flags().Uint64Var(&opts.PolicyGroup, "policy-group", 0, "Policy group ID")
	cmd.Flags().StringVar(&opts.ApplicationFile, "application-file", "", "Application JSON file or -")
	cmd.Flags().StringVar(&opts.DetectorConfigFile, "detector-config-file", "", "Detector config JSON file or -")
	cmd.Flags().StringVar(&opts.SessionFile, "session-file", "", "Session method JSON file or -")
	cmd.Flags().StringVar(&opts.AccessLogFile, "access-log-file", "", "Access log JSON file or -")
	cmd.Flags().StringVar(&opts.PayloadFile, "payload-file", "", "Full request payload JSON file or -")
	if typ == "" || isProxyObjectType(typ) {
		cmd.Flags().StringArrayVar(&opts.Listeners, "listener", nil, "Listener ID; repeatable or comma separated")
		cmd.Flags().Uint64Var(&opts.CertID, "cert-id", 0, "SSL certificate ID")
		cmd.Flags().StringVar(&opts.ProxyDetectionConfigFile, "proxy-detection-config-file", "", "Proxy detection config JSON file or -")
		cmd.Flags().StringVar(&opts.CustomNginxConfigFile, "custom-nginx-config-file", "", "Custom nginx config JSON file or -")
	}
	if typ == "" || !isProxyObjectType(typ) {
		cmd.Flags().StringArrayVar(&opts.RemoteListeners, "remote-listener", nil, "Remote listener IP:PORT or CIDR:PORT")
		cmd.Flags().StringVar(&opts.RemoteListenerFile, "remote-listener-file", "", "Remote listener JSON file or -")
	}
	if typ == "" || typ == "reverse-proxy" {
		cmd.Flags().StringVar(&opts.BackendType, "backend-type", "proxy", "Backend type (proxy|redirect)")
		cmd.Flags().StringArrayVar(&opts.Upstreams, "upstream", nil, "Upstream URL, e.g. http://10.0.0.1:8080,weight=2")
		cmd.Flags().StringVar(&opts.LoadBalance, "load-balance", "round-robin", "Load balance policy")
		cmd.Flags().BoolVar(&opts.BackendHTTP2, "backend-http2", false, "Use backend HTTP/2")
		cmd.Flags().BoolVar(&opts.BackendNTLM, "backend-ntlm", false, "Enable backend NTLM")
		cmd.Flags().StringVar(&opts.RedirectURL, "redirect-url", "", "Redirect URL")
		cmd.Flags().IntVar(&opts.RedirectCode, "redirect-code", 302, "Redirect status code")
	}
	addWriteFlags(cmd, &opts.writeOptions)
}

func runSiteCreate(cmd *cobra.Command, typ string, opts siteCreateOptions) error {
	typ, err := normalizeType(typ)
	if err != nil {
		return err
	}
	var body any
	if opts.PayloadFile != "" {
		body, err = readJSONFile(opts.PayloadFile)
		if err != nil {
			return err
		}
	} else {
		payload, err := buildSitePayload(cmd, typ, opts)
		if err != nil {
			return err
		}
		body = payload
	}
	path, _ := pathForType(typ)
	return doWrite(cmd, opts.writeOptions, "site.create."+typ, http.MethodPost, path, nil, body, nil)
}

func buildSitePayload(cmd *cobra.Command, typ string, opts siteCreateOptions) (map[string]any, error) {
	if opts.NodeGroup == 0 {
		return nil, fmt.Errorf("--node-group is required")
	}
	group, err := fetchNodeGroup(cmd, opts.NodeGroup)
	if err != nil {
		return nil, err
	}
	mode := stringField(group, "mode")
	if !isTypeSupportedByMode(typ, mode) {
		return nil, fmt.Errorf("node group %d is mode %s, cannot create %s; supported types: %s", opts.NodeGroup, mode, typ, strings.Join(supportedSiteTypes(mode), ", "))
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	domains := splitValues(opts.Domains)
	if len(domains) == 0 {
		return nil, fmt.Errorf("--domain is required")
	}
	payload := map[string]any{
		"node_group":   opts.NodeGroup,
		"name":         opts.Name,
		"domain_names": domains,
		"is_enabled":   opts.Enabled,
		"comment":      opts.Comment,
		"applications": []any{},
	}
	if isProxyObjectType(typ) {
		listeners, err := parseUintList(opts.Listeners)
		if err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, fmt.Errorf("--listener is required for %s", typ)
		}
		payload["listeners"] = listeners
		if opts.CertID != 0 {
			payload["ssl_certificate"] = opts.CertID
		}
		if opts.ProxyDetectionConfigFile != "" {
			payload["proxy_detection_config"], err = readJSONFile(opts.ProxyDetectionConfigFile)
			if err != nil {
				return nil, err
			}
		}
		if opts.CustomNginxConfigFile != "" {
			payload["custom_nginx_config"], err = readJSONFile(opts.CustomNginxConfigFile)
			if err != nil {
				return nil, err
			}
		}
	} else {
		remote, err := buildRemoteListeners(opts)
		if err != nil {
			return nil, err
		}
		if len(remote) == 0 {
			return nil, fmt.Errorf("--remote-listener is required for %s", typ)
		}
		payload["listeners"] = remote
	}
	apps, err := buildApplications(typ, opts)
	if err != nil {
		return nil, err
	}
	payload["applications"] = apps
	return payload, nil
}

func buildApplications(typ string, opts siteCreateOptions) ([]any, error) {
	if opts.ApplicationFile != "" {
		value, err := readJSONFile(opts.ApplicationFile)
		if err != nil {
			return nil, err
		}
		switch v := value.(type) {
		case []any:
			return v, nil
		default:
			return []any{v}, nil
		}
	}
	state, err := stateValue(opts.State)
	if err != nil {
		return nil, err
	}
	op, err := urlPathOp(opts.URLPathOp)
	if err != nil {
		return nil, err
	}
	paths := splitValues(opts.URLPaths)
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	urlPaths := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		urlPaths = append(urlPaths, map[string]any{"path": path, "op": op})
	}
	app := map[string]any{
		"name":            opts.AppName,
		"is_default":      true,
		"url_paths":       urlPaths,
		"protected_state": state,
	}
	if opts.PolicyGroup != 0 {
		app["policy_group"] = opts.PolicyGroup
	}
	var errRead error
	if opts.DetectorConfigFile != "" {
		app["detector_config"], errRead = readJSONFile(opts.DetectorConfigFile)
		if errRead != nil {
			return nil, errRead
		}
	}
	if opts.SessionFile != "" {
		app["session_method"], errRead = readJSONFile(opts.SessionFile)
		if errRead != nil {
			return nil, errRead
		}
	}
	if opts.AccessLogFile != "" {
		app["access_log"], errRead = readJSONFile(opts.AccessLogFile)
		if errRead != nil {
			return nil, errRead
		}
	}
	if typ == "reverse-proxy" {
		action, err := buildReverseProxyAction(opts)
		if err != nil {
			return nil, err
		}
		app["reverse_proxy_action"] = action
	}
	return []any{app}, nil
}

func buildReverseProxyAction(opts siteCreateOptions) (map[string]any, error) {
	switch opts.BackendType {
	case "", "proxy":
		protocol, servers, err := parseUpstreams(opts.Upstreams)
		if err != nil {
			return nil, err
		}
		if len(servers) == 0 {
			return nil, fmt.Errorf("--upstream is required for --backend-type proxy")
		}
		return map[string]any{
			"action_type": "backend",
			"backend_config": map[string]any{
				"protocol":              protocol,
				"servers":               servers,
				"http2":                 opts.BackendHTTP2,
				"ntlm":                  opts.BackendNTLM,
				"load_balancing_config": map[string]any{"method": normalizeLoadBalance(opts.LoadBalance)},
			},
		}, nil
	case "redirect":
		if opts.RedirectURL == "" {
			return nil, fmt.Errorf("--redirect-url is required for --backend-type redirect")
		}
		return map[string]any{
			"action_type":   "redirect",
			"return_config": map[string]any{"code": opts.RedirectCode, "redirect": opts.RedirectURL},
		}, nil
	default:
		return nil, fmt.Errorf("--backend-type must be proxy or redirect")
	}
}

func parseUpstreams(values []string) (string, []map[string]any, error) {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			items = append(items, value)
		}
	}
	var protocol string
	servers := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rawURL, meta, _ := strings.Cut(item, ",")
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "", nil, fmt.Errorf("--upstream must be a URL such as http://10.0.0.1:8080")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return "", nil, fmt.Errorf("--upstream protocol must be http or https")
		}
		if protocol == "" {
			protocol = u.Scheme
		} else if protocol != u.Scheme {
			return "", nil, fmt.Errorf("all --upstream values must use the same protocol")
		}
		port := 80
		if u.Scheme == "https" {
			port = 443
		}
		if u.Port() != "" {
			parsed, err := strconv.Atoi(u.Port())
			if err != nil || parsed <= 0 || parsed > 65535 {
				return "", nil, fmt.Errorf("invalid upstream port %q", u.Port())
			}
			port = parsed
		}
		weight := 1
		for _, part := range strings.Split(meta, ",") {
			key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
			if ok && key == "weight" {
				parsed, err := strconv.Atoi(value)
				if err != nil || parsed <= 0 {
					return "", nil, fmt.Errorf("invalid upstream weight %q", value)
				}
				weight = parsed
			}
		}
		servers = append(servers, map[string]any{"host": u.Hostname(), "port": port, "weight": weight})
	}
	return protocol, servers, nil
}

func buildRemoteListeners(opts siteCreateOptions) ([]any, error) {
	var result []any
	if opts.RemoteListenerFile != "" {
		value, err := readJSONFile(opts.RemoteListenerFile)
		if err != nil {
			return nil, err
		}
		if list, ok := value.([]any); ok {
			result = append(result, list...)
		} else {
			result = append(result, value)
		}
	}
	for _, item := range splitValues(opts.RemoteListeners) {
		host, portRaw, ok := strings.Cut(item, ":")
		if !ok {
			return nil, fmt.Errorf("--remote-listener must be IP:PORT or CIDR:PORT")
		}
		port, err := strconv.Atoi(portRaw)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid remote listener port %q", portRaw)
		}
		result = append(result, map[string]any{"ip": host, "port": port})
	}
	return result, nil
}

func fetchProtectedObject(cmd *cobra.Command, typ string, id uint64) (any, error) {
	path, err := pathForType(typ)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("id", strconv.FormatUint(id, 10))
	var value any
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, path, query, nil, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func isProxyObjectType(typ string) bool {
	typ, _ = normalizeType(typ)
	return typ == "reverse-proxy" || typ == "route-proxy" || typ == "transparent-proxy"
}

func createStrategyForType(typ string) string {
	if typ == "reverse-proxy" {
		return "semantic"
	}
	return "semantic_limited"
}

func normalizeLoadBalance(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ip-hash", "ip_hash":
		return "IPHash"
	case "least-conn", "least_conn":
		return "LeastConnections"
	case "consistent-hash", "consistent_hash", "hash":
		return "ConsistentHash"
	case "session-sticky", "session_sticky":
		return "SessionSticky"
	default:
		return "RoundRobin"
	}
}

func splitHostPort(raw string) (string, int, error) {
	host, portRaw, err := net.SplitHostPort(raw)
	if err != nil {
		host, portRaw, _ = strings.Cut(raw, ":")
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port in %q", raw)
	}
	return host, port, nil
}
