package safeline3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newACLCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "acl", Short: "Manage ACL templates and rules"}
	template := &cobra.Command{Use: "template", Short: "Manage ACL templates"}
	template.AddCommand(newACLTemplateListCommand())
	template.AddCommand(newACLTemplateGetCommand())
	template.AddCommand(newACLTemplateCreateCommand())
	template.AddCommand(newACLTemplateUpdateCommand())
	template.AddCommand(newACLTemplateEnableCommand("enable", true))
	template.AddCommand(newACLTemplateEnableCommand("disable", false))
	template.AddCommand(newACLTemplateDeleteCommand())
	rule := &cobra.Command{Use: "rule", Short: "Manage ACL generated rules"}
	rule.AddCommand(newACLRuleListCommand())
	rule.AddCommand(newACLRuleDeleteCommand())
	cmd.AddCommand(template, rule)
	return cmd
}

func newACLTemplateListCommand() *cobra.Command {
	var name, targetType, mode, enabled string
	var nodeGroup uint64
	var page, pageSize int
	cmd := &cobra.Command{Use: "list", Short: "List ACL templates", RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{"offset": (page - 1) * pageSize, "count": pageSize}
		if name != "" {
			body["name"] = equalCondition(name)
		}
		if targetType != "" {
			body["target_type"] = equalCondition(targetType)
		}
		if mode != "" {
			body["mode"] = equalCondition(mode)
		}
		if enabled != "" {
			value, err := strconv.ParseBool(enabled)
			if err != nil {
				return fmt.Errorf("--enabled must be true or false")
			}
			body["is_enabled"] = boolCondition(value)
		}
		if nodeGroup != 0 {
			body["node_group"] = equalCondition(nodeGroup)
		}
		return doRequest(cmd, http.MethodPost, "/api/v3/acl/acl-template/list", nil, body)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Name filter")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Target type")
	cmd.Flags().StringVar(&mode, "mode", "", "Mode manual|auto")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Enabled true|false")
	cmd.Flags().Uint64Var(&nodeGroup, "node-group", 0, "Node group ID")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newACLTemplateGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get ACL template", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		return doRequest(cmd, http.MethodPost, "/api/v3/acl/acl-template/list", nil, map[string]any{"id": equalCondition(id), "offset": 0, "count": 1})
	}}
}

func newACLTemplateCreateCommand() *cobra.Command {
	var name, mode, scope, targetType, actionType string
	var nodeGroups []string
	var period, limit, statusCode, expire int
	var policyRule, responseFile uint64
	var enabled bool
	var payload string
	var opts writeOptions
	cmd := &cobra.Command{Use: "create", Short: "Create ACL template", RunE: func(cmd *cobra.Command, args []string) error {
		var body any
		var err error
		if payload != "" {
			body, err = readJSONFile(payload)
			if err != nil {
				return err
			}
		} else {
			if name == "" || mode == "" || scope == "" || targetType == "" || actionType == "" || period == 0 || limit == 0 {
				return fmt.Errorf("--name, --mode, --scope, --target-type, --period, --limit and --action-type are required")
			}
			ngs, err := parseUintList(nodeGroups)
			if err != nil {
				return err
			}
			if len(ngs) == 0 {
				return fmt.Errorf("--node-group is required")
			}
			mode = normalizeACLMode(mode)
			scope = normalizeACLScope(scope)
			targetType = normalizeACLTargetType(targetType)
			match := map[string]any{"scope": scope, "limit": limit, "period": period, "target_type": targetType, "policy": map[string]any{}}
			if policyRule != 0 {
				match["policy"] = map[string]any{"policy_rule": policyRule}
			}
			actionType = normalizeACLActionType(actionType)
			body = map[string]any{
				"name":           name,
				"mode":           mode,
				"node_group_ids": ngs,
				"match_method":   match,
				"action_type":    actionType,
				"action":         buildACLAction(actionType, statusCode, responseFile),
				"expire_period":  expire,
				"is_enabled":     enabled,
			}
		}
		return doWrite(cmd, opts, "acl.template.create", http.MethodPost, "/api/v3/acl/acl-template", nil, body, nil)
	}}
	addACLTemplateMutationFlags(cmd, &opts)
	cmd.Flags().StringVar(&name, "name", "", "Template name")
	cmd.Flags().StringVar(&mode, "mode", "", "forbidden|dryrun")
	cmd.Flags().StringArrayVar(&nodeGroups, "node-group", nil, "Node group ID; repeatable or comma separated")
	cmd.Flags().StringVar(&scope, "scope", "", "all|url|url-prefix|policy-rule|hook-rule")
	cmd.Flags().StringVar(&targetType, "target-type", "", "cidr|session|fingerprint")
	cmd.Flags().IntVar(&period, "period", 0, "Period in seconds")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit")
	cmd.Flags().StringVar(&actionType, "action-type", "", "deny|dryrun-limit|rate-limit")
	cmd.Flags().Uint64Var(&policyRule, "policy-rule", 0, "Policy rule ID")
	cmd.Flags().IntVar(&statusCode, "status-code", 403, "Response status code")
	cmd.Flags().Uint64Var(&responseFile, "response-file", 0, "Response file ID")
	cmd.Flags().IntVar(&expire, "expire-period", 0, "Expire period seconds")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable template")
	cmd.Flags().StringVar(&payload, "payload-file", "", "Full payload JSON file or -")
	return cmd
}

func newACLTemplateUpdateCommand() *cobra.Command {
	var payload string
	var opts writeOptions
	cmd := &cobra.Command{Use: "update <id>", Short: "Update ACL template from payload file", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if payload == "" {
			return fmt.Errorf("--payload-file is required")
		}
		body, err := readJSONFile(payload)
		if err != nil {
			return err
		}
		if m, ok := body.(map[string]any); ok {
			m["id"] = id
		}
		return doWrite(cmd, opts, "acl.template.update", http.MethodPut, "/api/v3/acl/acl-template", nil, body, nil)
	}}
	cmd.Flags().StringVar(&payload, "payload-file", "", "Full payload JSON file or -")
	addACLTemplateMutationFlags(cmd, &opts)
	return cmd
}

func newACLTemplateEnableCommand(name string, enabled bool) *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{Use: name + " <id...>", Short: name + " ACL templates", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		return doWrite(cmd, opts, "acl.template."+name, http.MethodPut, "/api/v3/acl/acl-template/enable-status", nil, map[string]any{"ids": ids, "is_enabled": enabled}, nil)
	}}
	addACLTemplateMutationFlags(cmd, &opts)
	return cmd
}

func newACLTemplateDeleteCommand() *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{Use: "delete <id...>", Short: "Delete ACL templates", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		return doWrite(cmd, opts, "acl.template.delete", http.MethodDelete, "/api/v3/acl/acl-template", nil, map[string]any{"ids": ids}, nil)
	}}
	addACLTemplateMutationFlags(cmd, &opts)
	return cmd
}

func newACLRuleListCommand() *cobra.Command {
	var templateID uint64
	var page, pageSize int
	cmd := &cobra.Command{Use: "list", Short: "List ACL rules", RunE: func(cmd *cobra.Command, args []string) error {
		if templateID == 0 {
			return fmt.Errorf("--template-id is required")
		}
		q := url.Values{}
		q.Set("template_id", strconv.FormatUint(templateID, 10))
		addPagination(q, page, pageSize)
		return doRequest(cmd, http.MethodGet, "/api/v3/acl/acl-rule", q, nil)
	}}
	cmd.Flags().Uint64Var(&templateID, "template-id", 0, "ACL template ID")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newACLRuleDeleteCommand() *cobra.Command {
	var templateID uint64
	var all bool
	var opts writeOptions
	cmd := &cobra.Command{Use: "delete <id>", Short: "Delete ACL rule", Args: func(cmd *cobra.Command, args []string) error {
		if all {
			return cobra.NoArgs(cmd, args)
		}
		return cobra.ExactArgs(1)(cmd, args)
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if templateID == 0 {
			return fmt.Errorf("--template-id is required")
		}
		body := map[string]any{"template_id": templateID}
		if all {
			body["all"] = true
		} else {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			body["id"] = id
		}
		return doWrite(cmd, opts, "acl.rule.delete", http.MethodDelete, "/api/v3/acl/acl-rule", nil, body, nil)
	}}
	cmd.Flags().Uint64Var(&templateID, "template-id", 0, "ACL template ID")
	cmd.Flags().BoolVar(&all, "all", false, "Delete all generated rules in the template")
	addWriteFlags(cmd, &opts)
	return cmd
}

func addACLTemplateMutationFlags(cmd *cobra.Command, opts *writeOptions) {
	addWriteFlags(cmd, opts)
}

func normalizeACLActionType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dryrun-limit":
		return "dryrun_limit"
	case "rate-limit":
		return "rate_limit"
	default:
		return value
	}
}

func normalizeACLMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "forbidden", "block", "deny":
		return "forbidden"
	case "dryrun", "dry-run":
		return "dryrun"
	default:
		return value
	}
}

func normalizeACLScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all":
		return "All"
	case "url":
		return "Url"
	case "url-prefix", "urlprefix", "prefix":
		return "UrlPrefix"
	case "policy-rule", "policyrule":
		return "PolicyRule"
	case "hook-rule", "hookrule":
		return "HookRule"
	default:
		return value
	}
}

func normalizeACLTargetType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cidr", "ip":
		return "CIDR"
	case "session":
		return "Session"
	case "fingerprint":
		return "Fingerprint"
	default:
		return value
	}
}

func buildACLAction(actionType string, statusCode int, responseFile uint64) map[string]any {
	action := map[string]any{
		"forbidden": map[string]any{
			"action":      "response",
			"status_code": statusCode,
			"path":        "",
		},
	}
	if actionType == "rate_limit" || actionType == "dryrun_limit" {
		action["rate_limit"] = 1
		action["rate_period"] = 1
	}
	if responseFile != 0 {
		action["response_file"] = responseFile
	}
	return action
}
