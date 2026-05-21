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
