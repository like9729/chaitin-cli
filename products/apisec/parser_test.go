package apisec

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenerateRawCommands(t *testing.T) {
	api := loadMinimalOpenAPI(t)
	parser := NewParser(api, nil)

	commands, err := parser.GenerateRawCommands()
	if err != nil {
		t.Fatalf("GenerateRawCommands() error = %v", err)
	}

	application := findCommand(commands, "application-api")
	if application == nil {
		t.Fatalf("application-api command not generated")
	}
	getCmd := findCommand(application.Commands(), "get")
	if getCmd == nil {
		t.Fatalf("application-api get command not generated")
	}
	if getCmd.Flags().Lookup("page") == nil {
		t.Fatalf("page flag not generated")
	}
	if !strings.Contains(getCmd.Long, "Endpoint: GET /api/ApplicationAPI") {
		t.Fatalf("help missing endpoint: %s", getCmd.Long)
	}
	if !strings.Contains(getCmd.Long, "Operation ID: ApplicationAPI_get") {
		t.Fatalf("help missing operation ID: %s", getCmd.Long)
	}

	postCmd := findCommand(application.Commands(), "post")
	if postCmd == nil {
		t.Fatalf("application-api post command not generated")
	}
	if postCmd.Flags().Lookup("body") == nil {
		t.Fatalf("body flag not generated")
	}
	if postCmd.Flags().Lookup("body-file") == nil {
		t.Fatalf("body-file flag not generated")
	}
}

func TestGenerateSemanticCommands(t *testing.T) {
	api := loadMinimalOpenAPI(t)
	mapping := &CLIMapping{Commands: []MappedCommand{
		{
			Path:        []string{"asset", "app", "list"},
			OperationID: "ApplicationAPI_get",
			Short:       "List APISec applications",
			Long:        "List applications with pagination.",
			Examples:    []string{"chaitin-cli apisec asset app list --page 1"},
			Flags: map[string]MappedFlag{
				"page": {Name: "page-number", Description: "Page number to fetch."},
			},
		},
	}}
	parser := NewParser(api, mapping)

	commands, err := parser.GenerateSemanticCommands()
	if err != nil {
		t.Fatalf("GenerateSemanticCommands() error = %v", err)
	}
	asset := findCommand(commands, "asset")
	if asset == nil {
		t.Fatalf("asset command not generated")
	}
	app := findCommand(asset.Commands(), "app")
	if app == nil {
		t.Fatalf("asset app command not generated")
	}
	list := findCommand(app.Commands(), "list")
	if list == nil {
		t.Fatalf("asset app list command not generated")
	}
	if list.Short != "List APISec applications" {
		t.Fatalf("Short = %q, want mapped short", list.Short)
	}
	if !strings.Contains(list.Long, "Operation ID: ApplicationAPI_get") {
		t.Fatalf("Long missing operation ID: %s", list.Long)
	}
	flag := list.Flags().Lookup("page-number")
	if flag == nil {
		t.Fatalf("mapped page-number flag not generated")
	}
	if flag.Usage != "Page number to fetch." {
		t.Fatalf("flag usage = %q, want mapped description", flag.Usage)
	}
}

func loadMinimalOpenAPI(t *testing.T) *OpenAPI {
	t.Helper()
	data, err := os.ReadFile("testdata/openapi_minimal.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	api, err := parseOpenAPI(data)
	if err != nil {
		t.Fatalf("parseOpenAPI() error = %v", err)
	}
	return api
}

func findCommand(commands []*cobra.Command, use string) *cobra.Command {
	for _, cmd := range commands {
		if cmd.Use == use {
			return cmd
		}
	}
	return nil
}
