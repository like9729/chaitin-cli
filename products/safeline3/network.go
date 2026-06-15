package safeline3

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

func newNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Query SafeLine-3 network information",
		Long: `Query SafeLine-3 network information.

Provided commands:
  overview          GET /api/v3/network/overview
  links             GET /api/v3/network/link/list
  network-service   GET /api/v3/network/link/network_service/list
  vrrp              GET /api/v3/network/vrrp or /api/v3/network/vrrp/all

This CLI intentionally does not provide soft-bypass or hard-bypass commands.`,
	}
	cmd.AddCommand(newNetworkOverviewCommand())
	cmd.AddCommand(newNetworkLinksCommand())
	cmd.AddCommand(newNetworkServiceCommand())
	cmd.AddCommand(newNetworkVRRPCommand())
	return cmd
}

func newNetworkOverviewCommand() *cobra.Command {
	var nodeID string
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Show network interface overview",
		Long: `Show network interface overview.

API:
  GET /api/v3/network/overview

Required:
  --node-id string`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeID == "" {
				return fmt.Errorf("--node-id is required")
			}
			return doRequest(cmd, http.MethodGet, "/api/v3/network/overview", nodeIDQuery(nodeID), nil)
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	return cmd
}

func newNetworkLinksCommand() *cobra.Command {
	var nodeID string
	cmd := &cobra.Command{
		Use:   "links",
		Short: "List network links for a node",
		Long: `List network links for a node.

API:
  GET /api/v3/network/link/list

Required:
  --node-id string`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeID == "" {
				return fmt.Errorf("--node-id is required")
			}
			return doRequest(cmd, http.MethodGet, "/api/v3/network/link/list", nodeIDQuery(nodeID), nil)
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	return cmd
}

func newNetworkServiceCommand() *cobra.Command {
	var nodeID string
	cmd := &cobra.Command{
		Use:   "network-service",
		Short: "List network service bindings",
		Long: `List network service bindings.

API:
  GET /api/v3/network/link/network_service/list

Optional:
  --node-id string`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest(cmd, http.MethodGet, "/api/v3/network/link/network_service/list", nodeIDQuery(nodeID), nil)
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	return cmd
}

func newNetworkVRRPCommand() *cobra.Command {
	var nodeID string
	cmd := &cobra.Command{
		Use:   "vrrp",
		Short: "Show VRRP configuration",
		Long: `Show VRRP configuration.

API:
  GET /api/v3/network/vrrp/all
  GET /api/v3/network/vrrp?node_id=NODE_ID

Optional:
  --node-id string  When omitted, queries all nodes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeID == "" {
				return doRequest(cmd, http.MethodGet, "/api/v3/network/vrrp/all", nil, nil)
			}
			return doRequest(cmd, http.MethodGet, "/api/v3/network/vrrp", nodeIDQuery(nodeID), nil)
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	return cmd
}

func nodeIDQuery(nodeID string) url.Values {
	q := url.Values{}
	if nodeID != "" {
		q.Set("node_id", nodeID)
	}
	return q
}
