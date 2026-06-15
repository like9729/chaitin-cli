package safeline3

import (
	"fmt"

	"github.com/chaitin/chaitin-cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	runtimeCfg       Config
	runtimeInsecure  bool
	verbose          bool
	verboseSensitive bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "safeline-3",
		Short: "SafeLine-3 API CLI",
		Long: `SafeLine-3 CLI

Authentication uses the SafeLine-3 API-TOKEN HTTP header.

Config example:
  safeline-3:
    url: https://your-safeline3.example
    api_token: your-api-token

Command layers:
  Entity commands are the main interface: site, node-group, listener, log,
  system, network, policy-group, policy-rule, acl, ip-group and monitor.
  raw request is only an API escape hatch for debugging or unwrapped APIs.

Common examples:
  chaitin-cli safeline-3 node-group list
  chaitin-cli safeline-3 site capabilities --node-group 1 --output json
  chaitin-cli safeline-3 site list --type reverse-proxy --page 1 --page-size 20
  chaitin-cli safeline-3 log attack list --start -24h --page 1 --page-size 20
  chaitin-cli safeline-3 raw request GET /api/v3/license`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			applyRuntimeConfig(cmd)
			if runtimeCfg.URL == "" {
				return fmt.Errorf("URL is required (use --url or configure safeline-3.url / SAFELINE_3_URL)")
			}
			return nil
		},
	}

	cmd.PersistentFlags().String("url", "", "SafeLine-3 API URL")
	cmd.PersistentFlags().String("api-token", "", "SafeLine-3 API token sent as API-TOKEN header")
	cmd.PersistentFlags().StringP("output", "o", "table", "Output format (table|json)")
	cmd.PersistentFlags().BoolVar(&runtimeInsecure, "insecure", true, "Skip TLS certificate verification")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print request URL, headers, and body")
	cmd.PersistentFlags().BoolVar(&verboseSensitive, "verbose-sensitive", false, "Print sensitive values such as API tokens in verbose output")

	RegisterModules(cmd)
	return cmd
}

func ApplyRuntimeConfig(cmd *cobra.Command, cfg config.Raw, isDryRun bool) {
	productCfg, err := config.DecodeProduct[Config](cfg, "safeline-3")
	if err != nil {
		return
	}
	runtimeCfg = productCfg
	dryRun = isDryRun
}

func applyRuntimeConfig(cmd *cobra.Command) {
	if flag := lookupFlag(cmd, "url"); flag != nil && !flag.Changed && runtimeCfg.URL != "" {
		_ = setFlag(cmd, "url", runtimeCfg.URL)
	}
	if flag := lookupFlag(cmd, "api-token"); flag != nil && !flag.Changed && runtimeCfg.APIToken != "" {
		_ = setFlag(cmd, "api-token", runtimeCfg.APIToken)
	}
	if flag := lookupFlag(cmd, "url"); flag != nil {
		runtimeCfg.URL = flag.Value.String()
	}
	if flag := lookupFlag(cmd, "api-token"); flag != nil {
		runtimeCfg.APIToken = flag.Value.String()
	}
}

func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(name)
}

func setFlag(cmd *cobra.Command, name, value string) error {
	if cmd.Flags().Lookup(name) != nil {
		return cmd.Flags().Set(name, value)
	}
	if cmd.PersistentFlags().Lookup(name) != nil {
		return cmd.PersistentFlags().Set(name, value)
	}
	return cmd.InheritedFlags().Set(name, value)
}

func getClient(cmd *cobra.Command) *Client {
	return NewClient(runtimeCfg, runtimeInsecure, verbose)
}

func getRenderer(cmd *cobra.Command) Renderer {
	format := FormatJSON
	if flag := lookupFlag(cmd, "output"); flag != nil && flag.Value.String() == string(FormatTable) {
		format = FormatTable
	}
	return NewRenderer(format, cmd.OutOrStdout())
}
