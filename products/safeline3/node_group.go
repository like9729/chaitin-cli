package safeline3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type nodeGroupListResponse struct {
	Items []map[string]any `json:"items"`
	Total int              `json:"total"`
}

func newNodeGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node-group",
		Short: "Manage SafeLine-3 node groups",
		Long: `Manage SafeLine-3 node groups.

Protection object creation depends on the target node group's mode. Use
"node-group capabilities <id>" before creating listener or site resources.`,
	}
	cmd.AddCommand(newNodeGroupListCommand())
	cmd.AddCommand(newNodeGroupGetCommand())
	cmd.AddCommand(newNodeGroupNodesCommand())
	cmd.AddCommand(newNodeGroupNetworkCommand())
	cmd.AddCommand(newNodeGroupCapabilitiesCommand())
	cmd.AddCommand(newNodeGroupCreateCommand())
	cmd.AddCommand(newNodeGroupUpdateCommand())
	cmd.AddCommand(newNodeGroupSetModeCommand())
	cmd.AddCommand(newNodeGroupDeleteCommand())
	return cmd
}

func newNodeGroupListCommand() *cobra.Command {
	var mode, name, id string
	var onlyDefault, standalone bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List node groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			var res nodeGroupListResponse
			if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/node_group/list", nil, nil, &res); err != nil {
				return err
			}
			if mode != "" {
				normalized, err := normalizeMode(mode)
				if err != nil {
					return err
				}
				res.Items = filterMaps(res.Items, func(item map[string]any) bool { return stringField(item, "mode") == normalized })
			}
			if name != "" {
				res.Items = filterMaps(res.Items, func(item map[string]any) bool { return containsFold(stringField(item, "name"), name) })
			}
			if id != "" {
				res.Items = filterMaps(res.Items, func(item map[string]any) bool { return fmt.Sprint(item["id"]) == id })
			}
			if onlyDefault {
				res.Items = filterMaps(res.Items, func(item map[string]any) bool { return boolField(item, "is_default") })
			}
			if standalone {
				res.Items = filterMaps(res.Items, func(item map[string]any) bool { return boolField(item, "standalone") })
			}
			for _, item := range res.Items {
				mode := stringField(item, "mode")
				item["supported_site_types"] = supportedSiteTypes(mode)
			}
			res.Total = len(res.Items)
			return getRenderer(cmd).Render(res)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "", "Filter by node group mode (reverse-proxy|route-proxy|transparent|mirror|sdk)")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name")
	cmd.Flags().StringVar(&id, "id", "", "Filter by ID")
	cmd.Flags().BoolVar(&onlyDefault, "default", false, "Only show default node groups")
	cmd.Flags().BoolVar(&standalone, "standalone", false, "Only show standalone node groups")
	return cmd
}

func newNodeGroupGetCommand() *cobra.Command {
	var withNodes, withNetwork bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a node group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			group, err := fetchNodeGroup(cmd, id)
			if err != nil {
				return err
			}
			out := map[string]any{"node_group": group, "capabilities": capabilitiesForNodeGroup(group)}
			if withNodes {
				nodes, err := fetchNodeGroupNodes(cmd, id, 1, 200)
				if err != nil {
					return err
				}
				out["nodes"] = nodes
			}
			if withNetwork {
				network, err := fetchNodeGroupNetwork(cmd, id)
				if err != nil {
					return err
				}
				out["network"] = network
			}
			return getRenderer(cmd).Render(out)
		},
	}
	cmd.Flags().BoolVar(&withNodes, "with-nodes", false, "Include node list")
	cmd.Flags().BoolVar(&withNetwork, "with-network", false, "Include network summary")
	return cmd
}

func newNodeGroupNodesCommand() *cobra.Command {
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "nodes <id>",
		Short: "List nodes in a node group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			res, err := fetchNodeGroupNodes(cmd, id, page, pageSize)
			if err != nil {
				return err
			}
			return getRenderer(cmd).Render(res)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newNodeGroupNetworkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "network <id>",
		Short: "Show node group network summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			res, err := fetchNodeGroupNetwork(cmd, id)
			if err != nil {
				return err
			}
			return getRenderer(cmd).Render(res)
		},
	}
}

func newNodeGroupCapabilitiesCommand() *cobra.Command {
	var typ string
	cmd := &cobra.Command{
		Use:   "capabilities <id>",
		Short: "Show site and listener capabilities for a node group",
		Long: `Show site and listener capabilities for a node group.

API:
  GET /api/v3/node_group/list, then local compatibility calculation.

Required:
  <id> positive integer node group ID

Optional:
  --type reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk

Output:
  Node group mode, supported protected object types, listener types, and notes
  for site create/listener create decisions.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			group, err := fetchNodeGroup(cmd, id)
			if err != nil {
				return err
			}
			caps := capabilitiesForNodeGroup(group)
			if typ != "" {
				normalized, err := normalizeType(typ)
				if err != nil {
					return err
				}
				caps["type"] = normalized
				caps["supported"] = isTypeSupportedByMode(normalized, stringField(group, "mode"))
			}
			return getRenderer(cmd).Render(caps)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Protected object type")
	return cmd
}

func newNodeGroupCreateCommand() *cobra.Command {
	var name, mode string
	var nodeValues []string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a node group",
		Long: `Create a node group.

API:
  POST /api/v3/node_group

Required:
  --name string
  --mode reverse-proxy|route-proxy|transparent|mirror
  --yes

Behavior:
  SafeLine-3 backend only allows node group create in hardware cluster
  environments. Single-node default node groups are product-owned and cannot
  be created, updated or deleted by this command.

SDK note:
  SDK node groups are created by SDK/software deployment flows. The workmode
  and node-group create APIs do not create SDK mode groups.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			apiMode, err := normalizeMode(mode)
			if err != nil {
				return err
			}
			if apiMode == "SDK" {
				return fmt.Errorf("node-group create does not create SDK mode groups; SDK is only available when the backend already returns a node group in SDK mode")
			}
			nodes, err := parseUintList(nodeValues)
			if err != nil {
				return err
			}
			body := map[string]any{"name": name, "mode": apiMode, "nodes": nodes, "gm_supported": false}
			return doWrite(cmd, opts, "node-group.create", http.MethodPost, "/api/v3/node_group", nil, body, []string{"gm_supported is fixed to false in the first CLI version"})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Node group name")
	cmd.Flags().StringVar(&mode, "mode", "", "Node group mode (reverse-proxy|route-proxy|transparent|mirror)")
	cmd.Flags().StringArrayVar(&nodeValues, "node", nil, "Initial node numeric ID; repeatable or comma separated")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newNodeGroupUpdateCommand() *cobra.Command {
	var name string
	var addNodes, removeNodes []string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update node group name or members",
		Long: `Update node group name or members.

API:
  PUT /api/v3/node_group

Required:
  <id> positive integer
  --yes

Behavior:
  SafeLine-3 backend only allows node group update in hardware cluster
  environments. The default node group in single-node deployments is
  product-owned and cannot be edited; use node-group set-mode for supported
  work mode changes instead.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			group, err := fetchNodeGroup(cmd, id)
			if err != nil {
				return err
			}
			if boolField(group, "is_default") {
				return fmt.Errorf("default node group %d cannot be updated; use node-group set-mode for supported work mode changes", id)
			}
			add, err := parseUintList(addNodes)
			if err != nil {
				return err
			}
			remove, err := parseUintList(removeNodes)
			if err != nil {
				return err
			}
			if name == "" {
				name = stringField(group, "name")
			}
			body := map[string]any{"id": id, "name": name, "mode": stringField(group, "mode"), "add_nodes": add, "delete_nodes": remove}
			return doWrite(cmd, opts, "node-group.update", http.MethodPut, "/api/v3/node_group", nil, body, nil)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New node group name")
	cmd.Flags().StringArrayVar(&addNodes, "add-node", nil, "Node numeric ID to add")
	cmd.Flags().StringArrayVar(&removeNodes, "remove-node", nil, "Node numeric ID to remove")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newNodeGroupSetModeCommand() *cobra.Command {
	var mode string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "set-mode <id>",
		Short: "Change node group work mode",
		Long:  "Change node group work mode. This is high risk: the backend deletes listeners in the node group and resets/reinitializes network configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			apiMode, err := normalizeMode(mode)
			if err != nil {
				return err
			}
			if apiMode == "SDK" {
				return fmt.Errorf("workmode API does not accept SDK mode")
			}
			body := map[string]any{"node_group_id": id, "mode": apiMode}
			notes := []string{"Backend deletes listeners in this node group and resets/reinitializes node network configuration."}
			return doWrite(cmd, opts, "node-group.set-mode", http.MethodPut, "/api/v3/workmode", nil, body, notes)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "", "New mode (reverse-proxy|route-proxy|transparent|mirror)")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newNodeGroupDeleteCommand() *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "delete <id...>",
		Short: "Delete node groups",
		Long: `Delete node groups.

API:
  DELETE /api/v3/node_group

Required:
  <id...> one or more positive integer IDs
  --yes

Behavior:
  SafeLine-3 backend only allows node group delete in hardware cluster
  environments. Default node groups are product-owned and cannot be deleted.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDs(args)
			if err != nil {
				return err
			}
			if opts.Check {
				checks := make([]any, 0, len(ids))
				for _, id := range ids {
					nodes, _ := fetchNodeGroupNodes(cmd, id, 1, 10)
					checks = append(checks, map[string]any{"id": id, "nodes": nodes})
				}
				return getRenderer(cmd).Render(map[string]any{"operation": "node-group.delete", "dry_run": true, "checks": checks})
			}
			for _, id := range ids {
				group, err := fetchNodeGroup(cmd, id)
				if err != nil {
					return err
				}
				if boolField(group, "is_default") {
					return fmt.Errorf("default node group %d cannot be deleted", id)
				}
				body := map[string]any{"id": id}
				if err := doWrite(cmd, writeOptions{Yes: opts.Yes, Explain: opts.Explain}, "node-group.delete", http.MethodDelete, "/api/v3/node_group", nil, body, nil); err != nil {
					return err
				}
			}
			return nil
		},
	}
	addWriteFlags(cmd, &opts)
	return cmd
}

func fetchNodeGroup(cmd *cobra.Command, id uint64) (map[string]any, error) {
	res, err := fetchNodeGroups(cmd)
	if err != nil {
		return nil, err
	}
	for _, item := range res.Items {
		if fmt.Sprint(item["id"]) == strconv.FormatUint(id, 10) {
			return item, nil
		}
	}
	return nil, fmt.Errorf("node group %d not found", id)
}

func fetchNodeGroups(cmd *cobra.Command) (nodeGroupListResponse, error) {
	var res nodeGroupListResponse
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/node_group/list", nil, nil, &res); err != nil {
		return res, err
	}
	return res, nil
}

func fetchNodeGroupNodes(cmd *cobra.Command, id uint64, page, pageSize int) (any, error) {
	query := url.Values{}
	query.Set("id", strconv.FormatUint(id, 10))
	addPagination(query, page, pageSize)
	var res any
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/node_group/nodes", query, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func fetchNodeGroupNetwork(cmd *cobra.Command, id uint64) (any, error) {
	query := url.Values{}
	query.Set("id", strconv.FormatUint(id, 10))
	var res any
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/node_group/network/summary", query, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func capabilitiesForNodeGroup(group map[string]any) map[string]any {
	mode := stringField(group, "mode")
	return map[string]any{
		"node_group":           group["id"],
		"mode":                 modeCLI(mode),
		"supported_site_types": supportedSiteTypes(mode),
		"listener_types":       listenerTypesForMode(mode),
	}
}

func listenerTypesForMode(mode string) []string {
	switch mode {
	case "ReverseProxy":
		return []string{"reverse-proxy"}
	case "RouteProxy":
		return []string{"route-proxy"}
	case "Transparent":
		return []string{"transparent-proxy"}
	default:
		return nil
	}
}
