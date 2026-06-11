package dsensor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chaitin/chaitin-cli/config"
	"github.com/chaitin/chaitin-cli/products/dsensor/internal/parser"
	"gopkg.in/yaml.v3"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	for _, name := range []string{"url", "api-key", "refresh-cache", "output", "insecure", "verbose"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("missing persistent flag --%s", name)
		}
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute help error = %v", err)
	}
	help := out.String()
	for _, want := range []string{"D-Sensor CLI", "agent", "honeypot", "alarm"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestApplyRuntimeConfig(t *testing.T) {
	oldDryRun := dryRun
	t.Cleanup(func() {
		dryRun = oldDryRun
		runtimeCfg = Config{}
		urlFlag = ""
		apiKeyFlag = ""
	})

	cmd := NewCommand()
	cfg := config.Raw{}
	var node yaml.Node
	if err := node.Encode(Config{URL: "https://dsensor.example", APIKey: "token-1"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	cfg["dsensor"] = node

	ApplyRuntimeConfig(cmd, cfg, true)
	cmd.PersistentPreRun(cmd, nil)

	if got, _ := cmd.PersistentFlags().GetString("url"); got != "https://dsensor.example" {
		t.Fatalf("url = %q, want config value", got)
	}
	if got, _ := cmd.PersistentFlags().GetString("api-key"); got != "token-1" {
		t.Fatalf("api-key = %q, want config value", got)
	}
	if !dryRun {
		t.Fatalf("dryRun = false, want true")
	}
}

func TestDryRunRendersRequest(t *testing.T) {
	body := []byte(`{"sn":"agent-1"}`)
	out, err := renderDryRun(parserCommandSpec("POST", "/api/agent/detail"), body)
	if err != nil {
		t.Fatalf("renderDryRun() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("dry-run output is not JSON: %v\n%s", err, string(out))
	}
	if got["method"] != "POST" || got["path"] != "/api/agent/detail" {
		t.Fatalf("unexpected dry-run output: %#v", got)
	}
}

func TestDryRunDoesNotRequireURL(t *testing.T) {
	oldDryRun := dryRun
	oldURL := urlFlag
	t.Cleanup(func() {
		dryRun = oldDryRun
		urlFlag = oldURL
	})

	dryRun = true
	urlFlag = ""
	setupRunner()

	if parser.DefaultRunner == nil {
		t.Fatal("DefaultRunner is nil, want dry-run runner without URL")
	}
}

func parserCommandSpec(method, path string) parser.CommandSpec {
	return parser.CommandSpec{Method: method, Path: path}
}
