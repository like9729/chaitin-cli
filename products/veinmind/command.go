package veinmind

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/chaitin/chaitin-cli/config"
	"github.com/spf13/cobra"
)

//go:embed openapi.json
var openAPIFS embed.FS

var (
	runtimeCfg Config
	dryRun     bool
	verbose    bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "veinmind",
		Short: "VeinMind management tool",
		Long: `VeinMind CLI

面向容器安全场景的命令行工具，基于 VeinMind OpenAPI 动态生成命令。

`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			applyRuntimeConfig(cmd)
		},
	}

	cmd.PersistentFlags().String("url", "", "API URL")
	cmd.PersistentFlags().String("api-key", "", "API Key for authentication")
	cmd.PersistentFlags().StringP("output", "o", "table", "Output format (table|json)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print request URL, headers, and body")

	if err := loadDynamicCommands(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	return cmd
}

func ApplyRuntimeConfig(cmd *cobra.Command, cfg config.Raw, isDryRun bool) {
	productCfg, err := config.DecodeProduct[Config](cfg, "veinmind")
	if err != nil {
		return
	}
	runtimeCfg = productCfg
	dryRun = isDryRun
}

func applyRuntimeConfig(cmd *cobra.Command) {
	if flag := cmd.Flags().Lookup("url"); flag != nil && !flag.Changed && runtimeCfg.URL != "" {
		_ = cmd.Flags().Set("url", runtimeCfg.URL)
	}
	if flag := cmd.Flags().Lookup("api-key"); flag != nil && !flag.Changed && runtimeCfg.APIKey != "" {
		_ = cmd.Flags().Set("api-key", runtimeCfg.APIKey)
	}
}

func loadDynamicCommands(cmd *cobra.Command) error {
	data, err := openAPIFS.ReadFile("openapi.json")
	if err != nil {
		return fmt.Errorf("failed to read embedded openapi.json: %w", err)
	}

	var api OpenAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return fmt.Errorf("failed to parse openapi.json: %w", err)
	}

	parser := NewParser()
	commands, err := parser.GenerateCommands(&api)
	if err != nil {
		return fmt.Errorf("failed to generate commands: %w", err)
	}

	for _, command := range commands {
		cmd.AddCommand(command)
	}
	return nil
}

func getRenderer(cmd *cobra.Command) Renderer {
	format := FormatTable
	if output, _ := cmd.Flags().GetString("output"); output == "json" {
		format = FormatJSON
	}
	return NewRenderer(format, cmd.OutOrStdout())
}

func getClient(cmd *cobra.Command) *Client {
	url, _ := cmd.Flags().GetString("url")
	apiKey, _ := cmd.Flags().GetString("api-key")
	return NewClient(&Config{
		URL:    url,
		APIKey: apiKey,
	}, nil, verbose)
}
