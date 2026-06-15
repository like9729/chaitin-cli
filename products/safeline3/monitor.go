package safeline3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Query SafeLine-3 monitoring state",
		Long: `Query SafeLine-3 monitoring state.

Use node-group commands for node group management. "monitor node-groups" is
kept as a compatibility alias for read-only monitoring workflows.`,
	}
	cmd.AddCommand(newMonitorNodeGroupsCommand())
	cmd.AddCommand(simpleGetCommand("nodes", "Show current node states", "/api/v3/monitoring/node_state"))
	cmd.AddCommand(newMonitorNodeStateHistoryCommand("node-state-history", "/api/v3/monitoring/node_state/history", false))
	cmd.AddCommand(newMonitorNodeStateHistoryCommand("node-state-extended-history", "/api/v3/monitoring/node_state/extended_history", true))
	return cmd
}

func newMonitorNodeGroupsCommand() *cobra.Command {
	cmd := newNodeGroupListCommand()
	cmd.Use = "node-groups"
	cmd.Short = "List node groups (alias of node-group list)"
	cmd.Long = `List node groups.

Alias:
  monitor node-groups is a read-only compatibility alias of node-group list.

API:
  GET /api/v3/node_group/list`
	return cmd
}

func newMonitorNodeStateHistoryCommand(use, path string, extended bool) *cobra.Command {
	var nodeID, start, end string
	var metrics []string
	cmd := &cobra.Command{
		Use:   use,
		Short: "Query node state history",
		Long: fmt.Sprintf(`Query node state history.

API:
  GET %s

Required:
  --node-id string
  --start TIME
  --end TIME

Formats:
  --start/--end accept RFC3339, local time, Unix seconds/ms, relative values,
  or now. The backend expects timestamp_from and timestamp_to in Unix seconds.`, path),
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeID == "" {
				return fmt.Errorf("--node-id is required")
			}
			if start == "" {
				return fmt.Errorf("--start is required")
			}
			if end == "" {
				return fmt.Errorf("--end is required")
			}
			from, err := parseTimeValue(start)
			if err != nil {
				return err
			}
			to, err := parseTimeValue(end)
			if err != nil {
				return err
			}
			fromSeconds := from / 1000
			toSeconds := to / 1000
			if fromSeconds >= toSeconds {
				return fmt.Errorf("--start must be before --end")
			}
			q := url.Values{}
			q.Set("node_id", nodeID)
			q.Set("timestamp_from", strconv.FormatInt(fromSeconds, 10))
			q.Set("timestamp_to", strconv.FormatInt(toSeconds, 10))
			if extended {
				for _, metric := range splitValues(metrics) {
					q.Add("metrics", metric)
				}
			}
			return doRequest(cmd, http.MethodGet, path, q, nil)
		},
	}
	cmd.Flags().StringVar(&nodeID, "node-id", "", "Node ID")
	cmd.Flags().StringVar(&start, "start", "", "Start time")
	cmd.Flags().StringVar(&end, "end", "", "End time")
	if extended {
		cmd.Flags().StringArrayVar(&metrics, "metric", nil, "Metric name; repeatable or comma separated")
	}
	return cmd
}
