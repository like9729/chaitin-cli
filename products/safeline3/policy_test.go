package safeline3

import "testing"

func TestNormalizePolicyRuleAction(t *testing.T) {
	tests := map[string]string{
		"deny":               "deny",
		"block":              "deny",
		"allow":              "allow",
		"dry-run":            "dry_run",
		"modify-module":      "modify_module",
		"modify-skynet-rule": "modify_skynet_rule",
	}

	for input, expected := range tests {
		actual, err := normalizePolicyRuleAction(input)
		if err != nil {
			t.Fatalf("normalizePolicyRuleAction(%q) returned error: %v", input, err)
		}
		if actual != expected {
			t.Fatalf("normalizePolicyRuleAction(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSimplePolicyRuleConditionNormalizesAliases(t *testing.T) {
	condition := simplePolicyRuleCondition("src_ip", "in", []string{"10.0.0.1"})

	if condition["match_key"] != "remote_addr" {
		t.Fatalf("match_key = %v, want remote_addr", condition["match_key"])
	}
	if condition["operator"] != "cidr" {
		t.Fatalf("operator = %v, want cidr", condition["operator"])
	}
	if condition["custom_conflicts_group_id"] != 0 {
		t.Fatalf("custom_conflicts_group_id = %v, want 0", condition["custom_conflicts_group_id"])
	}
}
