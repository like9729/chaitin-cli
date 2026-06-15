package safeline3

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCommandRegistersExpectedModules(t *testing.T) {
	cmd := NewCommand()
	expected := []string{
		"acl",
		"ip-group",
		"listener",
		"log",
		"monitor",
		"network",
		"node-group",
		"policy-group",
		"policy-rule",
		"raw",
		"site",
		"system",
	}

	commands := map[string]bool{}
	for _, sub := range cmd.Commands() {
		commands[sub.Name()] = true
	}

	for _, name := range expected {
		if !commands[name] {
			t.Fatalf("missing command %q", name)
		}
	}
}

func TestImportantSafeline3HelpCommandsExist(t *testing.T) {
	root := NewCommand()
	tests := [][]string{
		{"node-group", "capabilities"},
		{"site", "create", "reverse-proxy"},
		{"listener", "create", "route-proxy"},
		{"listener", "update", "transparent-proxy"},
		{"policy-rule", "create", "simple"},
		{"system", "set-time"},
		{"network", "links"},
		{"log", "attack", "list"},
	}

	for _, path := range tests {
		cmd := root
		for _, name := range path {
			var nextFound bool
			for _, sub := range cmd.Commands() {
				if sub.Name() == name {
					cmd = sub
					nextFound = true
					break
				}
			}
			if !nextFound {
				t.Fatalf("missing command path %v at %q", path, name)
			}
		}
		if cmd.Short == "" {
			t.Fatalf("command path %v has empty short help", path)
		}
	}
}

func TestCommandLayersDoNotExposeShortCommandsYet(t *testing.T) {
	cmd := NewCommand()
	forbiddenRootCommands := []string{
		"sites",
		"node-groups",
		"attacks",
		"access-logs",
		"bot-logs",
		"time",
		"license",
		"protected-object",
		"log-event",
	}

	for _, name := range forbiddenRootCommands {
		if findSubcommand(cmd, name) != nil {
			t.Fatalf("root command %q should not be exposed in the current entity-command phase", name)
		}
	}
}

func TestForbiddenSafeline3FlagsAreNotExposed(t *testing.T) {
	cmd := NewCommand()
	forbiddenFlags := []string{
		"gm-cert-id",
		"protected-group",
		"tag",
	}

	walkCommands(cmd, func(c *cobra.Command) {
		for _, name := range forbiddenFlags {
			if c.Flags().Lookup(name) != nil || c.PersistentFlags().Lookup(name) != nil {
				t.Fatalf("command %q exposes forbidden flag --%s", c.CommandPath(), name)
			}
		}
	})
}

func TestLowLevelHTTPFlagsOnlyExistUnderRawRequest(t *testing.T) {
	cmd := NewCommand()
	rawRequest := findCommandPath(cmd, "raw", "request")
	if rawRequest == nil {
		t.Fatal("missing raw request command")
	}
	for _, name := range []string{"param", "body", "body-file"} {
		if rawRequest.Flags().Lookup(name) == nil {
			t.Fatalf("raw request missing --%s", name)
		}
	}
	for _, name := range []string{"query", "request", "request-file"} {
		if rawRequest.Flags().Lookup(name) != nil {
			t.Fatalf("raw request should not expose old alias --%s", name)
		}
	}

	lowLevelFlags := map[string]bool{
		"param":        true,
		"query":        true,
		"request":      true,
		"request-file": true,
		"body":         true,
		"body-file":    true,
	}
	walkCommands(cmd, func(c *cobra.Command) {
		if c == rawRequest || strings.HasPrefix(c.CommandPath(), rawRequest.CommandPath()) {
			return
		}
		for name := range lowLevelFlags {
			if c.Flags().Lookup(name) != nil || c.PersistentFlags().Lookup(name) != nil {
				t.Fatalf("non-raw command %q exposes low-level HTTP flag --%s", c.CommandPath(), name)
			}
		}
	})
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func findCommandPath(cmd *cobra.Command, path ...string) *cobra.Command {
	current := cmd
	for _, name := range path {
		current = findSubcommand(current, name)
		if current == nil {
			return nil
		}
	}
	return current
}

func walkCommands(cmd *cobra.Command, visit func(*cobra.Command)) {
	visit(cmd)
	for _, sub := range cmd.Commands() {
		walkCommands(sub, visit)
	}
}
