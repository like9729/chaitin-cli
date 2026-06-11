package parser

import (
	"testing"

	"github.com/chaitin/chaitin-cli/products/dsensor/internal/spec"
)

func TestParseSpec(t *testing.T) {
	s, err := ParseSpec(spec.SpecJSON)
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}
	if s.OpenAPI != "3.0.2" {
		t.Errorf("expected OpenAPI 3.0.2, got %s", s.OpenAPI)
	}
	if s.Info.Title != "产品 API 文档" {
		t.Errorf("expected title '产品 API 文档', got '%s'", s.Info.Title)
	}
	if len(s.Paths) != 257 {
		t.Errorf("expected 257 paths, got %d", len(s.Paths))
	}
}

func TestFlattenCommands(t *testing.T) {
	s, err := ParseSpec(spec.SpecJSON)
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	cmds, err := FlattenCommands(s)
	if err != nil {
		t.Fatalf("FlattenCommands failed: %v", err)
	}

	if len(cmds) != 257 {
		t.Errorf("expected 257 commands, got %d", len(cmds))
	}

	// Check all are POST
	for _, c := range cmds {
		if c.Method != "POST" {
			t.Logf("non-POST method found: %s %s -> %s", c.Method, c.Path, c.OperationID)
		}
		if c.OperationID == "" {
			t.Errorf("empty operationId for path: %s", c.Path)
		}
	}

	// Count unique tags
	tags := make(map[string]int)
	for _, c := range cmds {
		for _, tag := range c.Tags {
			tags[tag]++
		}
	}
	t.Logf("Unique tags (%d):", len(tags))
	for tag, count := range tags {
		mapped, ok := TagNameMap[tag]
		t.Logf("  %s (%s) -> %d operations [mapped: %s, known: %v]", tag, mapped, count, mapped, ok)
	}

	// Check body params extraction
	withBody := 0
	withParams := 0
	for _, c := range cmds {
		if c.HasBody {
			withBody++
		}
		if len(c.BodyParams) > 0 {
			withParams++
		}
	}
	t.Logf("Commands with body: %d, with flaggable params: %d", withBody, withParams)
}

func TestCommandName(t *testing.T) {
	tests := []struct {
		opID      string
		tagPrefix string
		expected  string
	}{
		{"agent_detail", "agent", "detail"},
		{"list_agent", "agent", "agent"},
		{"get_proto_cfg", "agent", "proto_cfg"},
		{"business_group_post", "agent", "business_group_post"}, // tagPrefix doesn't match, falls through to other prefixes
		{"start_honeypot", "honeypot", "honeypot"},              // starts with "start_"
		{"honey_pot_create", "honeypot", "honey_pot_create"},    // tag prefix doesn't match, falls through
	}
	for _, tt := range tests {
		got := CommandName(tt.opID, tt.tagPrefix)
		if got != tt.expected {
			t.Errorf("CommandName(%q, %q) = %q, want %q", tt.opID, tt.tagPrefix, got, tt.expected)
		}
	}
}

func TestBuildCommandTree(t *testing.T) {
	s, err := ParseSpec(spec.SpecJSON)
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	cmds, err := FlattenCommands(s)
	if err != nil {
		t.Fatalf("FlattenCommands failed: %v", err)
	}

	root := BuildCommandTree(cmds)
	if root == nil {
		t.Fatal("BuildCommandTree returned nil")
	}

	// Should have 13 tag-group parent commands
	children := root.Commands()
	if len(children) != len(TagNameMap) {
		t.Errorf("expected %d parent commands, got %d", len(TagNameMap), len(children))
	}

	// Check each parent has subcommands
	totalLeaves := 0
	for _, child := range children {
		leaves := child.Commands()
		if len(leaves) == 0 {
			t.Errorf("parent %q has no subcommands", child.Use)
		}
		totalLeaves += len(leaves)
		t.Logf("  %s (%s): %d subcommands", child.Use, child.Short, len(leaves))
	}

	if totalLeaves != 257 {
		t.Errorf("expected 257 total leaf commands, got %d", totalLeaves)
	}
}
