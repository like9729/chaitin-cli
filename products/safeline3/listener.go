package safeline3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type listenerOptions struct {
	writeOptions
	Name              string
	NodeGroup         uint64
	IP                string
	Port              int
	TLS               bool
	TLSProtocols      []string
	TLSCiphers        string
	HTTP2             bool
	NTLM              bool
	TransparentServer bool
	InboundIPs        []string
	TransparentPort   bool
	VirtualWirePair   string
	ProtectedObjects  []string
	PayloadFile       string
}

func newListenerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listener",
		Short: "Manage protected object listeners",
		Long: `Manage protected object listeners.

SafeLine-3 listener availability depends on node group mode. Create the
listener type that matches the target site type: reverse-proxy, route-proxy,
or transparent-proxy.`,
	}
	cmd.AddCommand(newListenerListCommand())
	cmd.AddCommand(newListenerCreateCommand())
	cmd.AddCommand(newListenerUpdateCommand())
	cmd.AddCommand(newListenerDeleteCommand())
	return cmd
}

func newListenerListCommand() *cobra.Command {
	var nodeGroup uint64
	var ip, tls string
	var port, page, pageSize int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List listeners",
		Long: `List protected object listeners.

API:
  POST /api/v3/protected-object/listener

Required:
  --node-group ID

Optional:
  --ip string
  --port int
  --tls true|false
  --page int
  --page-size int`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeGroup == 0 {
				return fmt.Errorf("--node-group is required")
			}
			body := map[string]any{"node_group": equalCondition(nodeGroup)}
			if ip != "" {
				body["ip"] = equalCondition(ip)
			}
			if port != 0 {
				body["port"] = equalCondition(port)
			}
			if tls != "" {
				parsed, err := strconv.ParseBool(tls)
				if err != nil {
					return fmt.Errorf("--tls must be true or false")
				}
				body["tls"] = equalCondition(parsed)
			}
			body["offset"] = (page - 1) * pageSize
			body["count"] = pageSize
			return doRequest(cmd, http.MethodPost, "/api/v3/protected-object/listener", nil, body)
		},
	}
	cmd.Flags().Uint64Var(&nodeGroup, "node-group", 0, "Node group ID")
	cmd.Flags().StringVar(&ip, "ip", "", "Listener IP/CIDR")
	cmd.Flags().IntVar(&port, "port", 0, "Listener port")
	cmd.Flags().StringVar(&tls, "tls", "", "TLS state true|false")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newListenerCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a listener",
		Long: `Create a protected object listener.

Use one subcommand per listener type:
  reverse-proxy       for ReverseProxy node groups
  route-proxy         for RouteProxy node groups
  transparent-proxy   for Transparent node groups`,
	}
	cmd.AddCommand(newListenerCreateTypedCommand("reverse-proxy"))
	cmd.AddCommand(newListenerCreateTypedCommand("route-proxy"))
	cmd.AddCommand(newListenerCreateTypedCommand("transparent-proxy"))
	return cmd
}

func newListenerCreateTypedCommand(typ string) *cobra.Command {
	var opts listenerOptions
	cmd := &cobra.Command{
		Use:   typ,
		Short: "Create a " + typ + " listener",
		Long:  listenerCreateLong(typ),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildListenerPayload(cmd, typ, opts, 0)
			if err != nil {
				return err
			}
			path, _ := listenerPathForType(typ)
			return doWrite(cmd, opts.writeOptions, "listener.create."+typ, http.MethodPost, path, nil, body, nil)
		},
	}
	addListenerFlags(cmd, &opts, typ)
	return cmd
}

func newListenerUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a listener",
		Long: `Update a protected object listener.

Use one subcommand per listener type:
  reverse-proxy       for ReverseProxy listeners
  route-proxy         for RouteProxy listeners
  transparent-proxy   for TransparentProxy listeners`,
	}
	cmd.AddCommand(newListenerUpdateTypedCommand("reverse-proxy"))
	cmd.AddCommand(newListenerUpdateTypedCommand("route-proxy"))
	cmd.AddCommand(newListenerUpdateTypedCommand("transparent-proxy"))
	return cmd
}

func newListenerUpdateTypedCommand(typ string) *cobra.Command {
	var opts listenerOptions
	cmd := &cobra.Command{
		Use:   typ + " <id>",
		Short: "Update a " + typ + " listener",
		Long:  listenerUpdateLong(typ),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			body, err := buildListenerPayload(cmd, typ, opts, id)
			if err != nil {
				return err
			}
			path, _ := listenerPathForType(typ)
			return doWrite(cmd, opts.writeOptions, "listener.update."+typ, http.MethodPut, path, nil, body, []string{"update sends a full listener payload"})
		},
	}
	addListenerFlags(cmd, &opts, typ)
	return cmd
}

func listenerCreateLong(typ string) string {
	switch typ {
	case "reverse-proxy":
		return `Create a reverse-proxy listener.

API:
  POST /api/v3/protected-object/reverse-proxy/listener

Required:
  --name string
  --node-group ID       Must be reverse-proxy mode.
  --ip IP/CIDR
  --port 1..65535
  --yes

Optional:
  --tls
  --tls-protocol string  Repeatable.
  --tls-ciphers string
  --http2               Requires --tls.
  --protected-object ID Repeatable or comma separated.
  --payload-file JSON   Full API payload.

Examples:
  chaitin-cli safeline-3 listener create reverse-proxy --name web-https --node-group 1 --ip 0.0.0.0 --port 443 --tls --http2 --yes
  chaitin-cli safeline-3 listener create reverse-proxy --payload-file listener.json --check`
	case "route-proxy":
		return `Create a route-proxy listener.

API:
  POST /api/v3/protected-object/route-proxy/listener

Required:
  --name string
  --node-group ID       Must be route-proxy mode.
  --ip IP/CIDR
  --port 1..65535
  --inbound-ip IP       Repeatable or comma separated.
  --yes

Optional:
  --tls
  --tls-protocol string  Repeatable.
  --tls-ciphers string
  --http2               Requires --tls.
  --ntlm
  --transparent-server
  --protected-object ID Repeatable or comma separated.
  --payload-file JSON   Full API payload.

Examples:
  chaitin-cli safeline-3 listener create route-proxy --name route-http --node-group 2 --ip 10.0.0.0/24 --port 80 --inbound-ip 10.0.0.10 --yes
  chaitin-cli safeline-3 listener create route-proxy --payload-file listener.json --check`
	case "transparent-proxy":
		return `Create a transparent-proxy listener.

API:
  POST /api/v3/protected-object/transparent-proxy/listener

Required:
  --name string
  --node-group ID       Must be transparent mode.
  --ip IP/CIDR
  --port 1..65535
  --virtual-wire-pair string
  --yes

Optional:
  --tls
  --tls-protocol string  Repeatable.
  --tls-ciphers string
  --http2               Requires --tls.
  --ntlm
  --transparent-port
  --protected-object ID Repeatable or comma separated.
  --payload-file JSON   Full API payload.

Examples:
  chaitin-cli safeline-3 listener create transparent-proxy --name tp-http --node-group 3 --ip 10.0.0.10 --port 80 --virtual-wire-pair wire0 --yes
  chaitin-cli safeline-3 listener create transparent-proxy --payload-file listener.json --check`
	default:
		return ""
	}
}

func listenerUpdateLong(typ string) string {
	path, _ := listenerPathForType(typ)
	return fmt.Sprintf(`Update a %s listener.

API:
  PUT %s

Required:
  <id> positive integer
  --name string
  --node-group ID
  --ip IP/CIDR
  --port 1..65535
  --yes

Behavior:
  The command sends a full listener payload. Use --payload-file for exact API
  payloads when the semantic flags are not enough.`, typ, path)
}

func newListenerDeleteCommand() *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "delete <id...>",
		Short: "Delete listeners",
		Long: `Delete listeners.

API:
  DELETE /api/v3/protected-object/listener

Required:
  <id...> one or more positive integer IDs
  --yes

Example:
  chaitin-cli safeline-3 listener delete 10 11 --check`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDs(args)
			if err != nil {
				return err
			}
			body := map[string]any{"ids": ids}
			return doWrite(cmd, opts, "listener.delete", http.MethodDelete, "/api/v3/protected-object/listener", nil, body, nil)
		},
	}
	addWriteFlags(cmd, &opts)
	return cmd
}

func addListenerFlags(cmd *cobra.Command, opts *listenerOptions, typ string) {
	cmd.Flags().StringVar(&opts.Name, "name", "", "Listener name")
	cmd.Flags().Uint64Var(&opts.NodeGroup, "node-group", 0, "Node group ID")
	cmd.Flags().StringVar(&opts.IP, "ip", "", "Listener IP/CIDR")
	cmd.Flags().IntVar(&opts.Port, "port", 0, "Listener port")
	cmd.Flags().BoolVar(&opts.TLS, "tls", false, "Enable TLS")
	cmd.Flags().StringArrayVar(&opts.TLSProtocols, "tls-protocol", nil, "TLS protocol; repeatable")
	cmd.Flags().StringVar(&opts.TLSCiphers, "tls-ciphers", "", "TLS ciphers")
	cmd.Flags().BoolVar(&opts.HTTP2, "http2", false, "Enable HTTP/2")
	if typ == "route-proxy" || typ == "transparent-proxy" {
		cmd.Flags().BoolVar(&opts.NTLM, "ntlm", false, "Enable backend NTLM")
	}
	if typ == "route-proxy" {
		cmd.Flags().BoolVar(&opts.TransparentServer, "transparent-server", false, "Enable transparent server")
		cmd.Flags().StringArrayVar(&opts.InboundIPs, "inbound-ip", nil, "Inbound IP; repeatable or comma separated")
	}
	if typ == "transparent-proxy" {
		cmd.Flags().BoolVar(&opts.TransparentPort, "transparent-port", false, "Enable transparent port")
		cmd.Flags().StringVar(&opts.VirtualWirePair, "virtual-wire-pair", "", "Virtual wire pair name")
	}
	cmd.Flags().StringArrayVar(&opts.ProtectedObjects, "protected-object", nil, "Protected object ID; repeatable or comma separated")
	cmd.Flags().StringVar(&opts.PayloadFile, "payload-file", "", "Full listener payload JSON file or -")
	addWriteFlags(cmd, &opts.writeOptions)
}

func buildListenerPayload(cmd *cobra.Command, typ string, opts listenerOptions, id uint64) (any, error) {
	if opts.PayloadFile != "" {
		return readJSONFile(opts.PayloadFile)
	}
	typ, err := normalizeListenerType(typ)
	if err != nil {
		return nil, err
	}
	if opts.NodeGroup == 0 {
		return nil, fmt.Errorf("--node-group is required")
	}
	group, err := fetchNodeGroup(cmd, opts.NodeGroup)
	if err != nil {
		return nil, err
	}
	if !isListenerTypeSupportedByMode(typ, stringField(group, "mode")) {
		return nil, fmt.Errorf("node group %d is mode %s, cannot use %s listener; supported listener types: %s", opts.NodeGroup, stringField(group, "mode"), typ, strings.Join(listenerTypesForMode(stringField(group, "mode")), ", "))
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	if opts.IP == "" {
		return nil, fmt.Errorf("--ip is required")
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return nil, fmt.Errorf("--port must be between 1 and 65535")
	}
	if !opts.TLS && (len(opts.TLSProtocols) > 0 || opts.TLSCiphers != "" || opts.HTTP2) {
		return nil, fmt.Errorf("--tls-protocol, --tls-ciphers and --http2 require --tls")
	}
	protectedObjects, err := parseUintList(opts.ProtectedObjects)
	if err != nil {
		return nil, err
	}
	inboundIPs := splitValues(opts.InboundIPs)
	if typ == "route-proxy" && len(inboundIPs) == 0 {
		return nil, fmt.Errorf("--inbound-ip is required for route-proxy listener")
	}
	if typ == "transparent-proxy" && opts.VirtualWirePair == "" {
		return nil, fmt.Errorf("--virtual-wire-pair is required for transparent-proxy listener")
	}
	body := map[string]any{
		"name":              opts.Name,
		"node_group":        opts.NodeGroup,
		"ip":                opts.IP,
		"port":              opts.Port,
		"tls":               opts.TLS,
		"tls_protocols":     splitValues(opts.TLSProtocols),
		"tls_ciphers":       opts.TLSCiphers,
		"http2":             opts.HTTP2,
		"protected_objects": protectedObjects,
	}
	switch typ {
	case "route-proxy":
		body["ntlm"] = opts.NTLM
		body["transparent_server"] = opts.TransparentServer
		body["inbound_ips"] = inboundIPs
	case "transparent-proxy":
		body["ntlm"] = opts.NTLM
		body["transparent_port"] = opts.TransparentPort
		body["virtual_wire_pair"] = opts.VirtualWirePair
	}
	if id != 0 {
		body["id"] = id
	}
	return body, nil
}

func normalizeListenerType(t string) (string, error) {
	switch t {
	case "reverse-proxy", "route-proxy", "transparent-proxy":
		return t, nil
	default:
		return "", fmt.Errorf("invalid listener type %q", t)
	}
}

func listenerPathForType(t string) (string, error) {
	t, err := normalizeListenerType(t)
	if err != nil {
		return "", err
	}
	return "/api/v3/protected-object/" + t + "/listener", nil
}

func isListenerTypeSupportedByMode(t, mode string) bool {
	t, _ = normalizeListenerType(t)
	for _, supported := range listenerTypesForMode(mode) {
		if supported == t {
			return true
		}
	}
	return false
}

func listenerQuery(nodeGroup uint64) url.Values {
	q := url.Values{}
	q.Set("node_group", strconv.FormatUint(nodeGroup, 10))
	return q
}
