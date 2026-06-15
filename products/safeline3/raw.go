package safeline3

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func newRawCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw",
		Short: "Raw SafeLine-3 API requests",
		Long: `Raw SafeLine-3 API requests.

Use raw only for APIs that are not yet wrapped by entity commands, or for
debugging exact request payloads. Normal workflows should prefer entity
commands such as site, node-group, listener, log, system and network.`,
	}
	cmd.AddCommand(newRawRequestCommand())
	return cmd
}

func newRawRequestCommand() *cobra.Command {
	var opts requestOptions
	cmd := &cobra.Command{
		Use:   "request METHOD PATH",
		Short: "Send an arbitrary SafeLine-3 API request",
		Long: `Send an arbitrary SafeLine-3 API request.

API:
  METHOD PATH

Required:
  METHOD  GET|POST|PUT|PATCH|DELETE
  PATH    /api/v3/... or a full URL

Optional:
  --param key=value      Query parameter; repeatable.
  --body JSON           JSON request body.
  --body-file PATH|-    JSON request body file or stdin.

Rules:
  --body and --body-file are mutually exclusive.

Examples:
  chaitin-cli safeline-3 raw request GET /api/v3/license
  chaitin-cli safeline-3 raw request POST /api/v3/protected-logger/DetectLogList --body-file body.json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			default:
				return fmt.Errorf("unsupported method %q", args[0])
			}

			query, err := parseQuery(opts.Query)
			if err != nil {
				return err
			}
			body, err := readRequestBody(opts)
			if err != nil {
				return err
			}

			var result any
			if err := getClient(cmd).Do(context.Background(), method, args[1], query, body, &result); err != nil {
				if _, ok := err.(dryRunResult); ok {
					return nil
				}
				return err
			}
			return getRenderer(cmd).Render(result)
		},
	}
	addRequestFlags(cmd, &opts, true, true)
	return cmd
}
