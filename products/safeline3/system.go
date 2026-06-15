package safeline3

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newSystemCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Query and update SafeLine-3 system information",
		Long: `Query and update SafeLine-3 system information.

Provided commands:
  license        GET /api/v3/license
  license-check  GET /api/v3/license/check
  machine-ids    GET /api/v3/management/machine_ids
  time           GET /api/v3/system/time
  set-time       PUT /api/v3/system/time

This CLI intentionally does not provide reboot or shutdown commands.`,
	}
	cmd.AddCommand(simpleGetCommand("license", "Show license information", "/api/v3/license"))
	cmd.AddCommand(simpleGetCommand("license-check", "Check license state", "/api/v3/license/check"))
	cmd.AddCommand(simpleGetCommand("machine-ids", "Show management machine IDs", "/api/v3/management/machine_ids"))
	cmd.AddCommand(simpleGetCommand("time", "Show management node system time", "/api/v3/system/time"))
	cmd.AddCommand(newSystemSetTimeCommand())
	return cmd
}

func newSystemSetTimeCommand() *cobra.Command {
	var timeValue string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "set-time",
		Short: "Set system time on all known nodes",
		Long: `Set system time on all known nodes.

API:
  PUT /api/v3/system/time

Required:
  --time TIME
  --yes

Formats:
  --time accepts RFC3339, local time "YYYY-MM-DD HH:MM:SS", Unix seconds,
  Unix milliseconds, relative values like -1h, or now.

Behavior:
  SafeLine-3 backend sets all known nodes. There is no --node-id parameter.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if timeValue == "" {
				return fmt.Errorf("--time is required")
			}
			timestamp, err := parseTimeValue(timeValue)
			if err != nil {
				return err
			}
			body := map[string]any{"timestamp": timestamp}
			return doWrite(cmd, opts, "system.set-time", http.MethodPut, "/api/v3/system/time", nil, body, []string{"backend sets time for all known nodes"})
		},
	}
	cmd.Flags().StringVar(&timeValue, "time", "", "Target time (RFC3339, local time, unix seconds/ms, relative like -1h, now)")
	addWriteFlags(cmd, &opts)
	return cmd
}

func simpleGetCommand(use, short, path string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long: fmt.Sprintf(`%s.

API:
  GET %s

Required:
  None.`, short, path),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest(cmd, http.MethodGet, path, nil, nil)
		},
	}
}
