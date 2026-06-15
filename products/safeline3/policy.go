package safeline3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newPolicyGroupCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "policy-group", Short: "Manage policy groups"}
	cmd.AddCommand(newPolicyGroupListCommand())
	cmd.AddCommand(newPolicyGroupAllCommand())
	cmd.AddCommand(newPolicyGroupGetCommand())
	cmd.AddCommand(newPolicyGroupCreateCommand())
	cmd.AddCommand(newPolicyGroupRenameCommand())
	cmd.AddCommand(newPolicyGroupModuleCommand())
	cmd.AddCommand(newPolicyGroupDeleteCommand())
	return cmd
}

func newPolicyGroupListCommand() *cobra.Command {
	var name, id string
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List policy groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"offset": (page - 1) * pageSize, "count": pageSize}
			if name != "" {
				body["name"] = equalCondition(name)
			}
			if id != "" {
				parsed, err := parseID(id)
				if err != nil {
					return err
				}
				body["id"] = equalCondition(parsed)
			}
			return doRequest(cmd, http.MethodPost, "/api/v3/detect/PolicyGroup/list", nil, body)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name filter")
	cmd.Flags().StringVar(&id, "id", "", "ID filter")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newPolicyGroupAllCommand() *cobra.Command {
	return &cobra.Command{Use: "all", Short: "List all policy groups", RunE: func(cmd *cobra.Command, args []string) error {
		return doRequest(cmd, http.MethodGet, "/api/v3/detect/PolicyGroup/all", nil, nil)
	}}
}

func newPolicyGroupGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get policy group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		return doRequest(cmd, http.MethodGet, "/api/v3/detect/PolicyGroup/detail", url.Values{"id": {strconv.FormatUint(id, 10)}}, nil)
	}}
}

func newPolicyGroupCreateCommand() *cobra.Command {
	var name, comment string
	var templateID uint64
	var opts writeOptions
	cmd := &cobra.Command{Use: "create", Short: "Create policy group", RunE: func(cmd *cobra.Command, args []string) error {
		if name == "" || templateID == 0 {
			return fmt.Errorf("--name and --template-id are required")
		}
		body := map[string]any{"name": name, "template_id": templateID, "comment": comment}
		return doWrite(cmd, opts, "policy-group.create", http.MethodPost, "/api/v3/detect/PolicyGroup", nil, body, nil)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Policy group name")
	cmd.Flags().Uint64Var(&templateID, "template-id", 0, "Template policy group ID")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyGroupRenameCommand() *cobra.Command {
	var name, comment string
	var opts writeOptions
	cmd := &cobra.Command{Use: "rename <id>", Short: "Rename policy group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		current, err := fetchPolicyGroup(cmd, id)
		if err != nil {
			return err
		}
		if comment == "" {
			comment = stringField(current, "comment")
		}
		body := map[string]any{
			"id":                            id,
			"name":                          name,
			"comment":                       comment,
			"modules_detection_config":      current["modules_detection_config"],
			"base_modules_detection_config": current["base_modules_detection_config"],
		}
		return doWrite(cmd, opts, "policy-group.rename", http.MethodPut, "/api/v3/detect/PolicyGroup", nil, body, []string{"SafeLine-3 policy-group update requires full module config; CLI reads current detail and preserves it"})
	}}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&comment, "comment", "", "New comment")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyGroupModuleCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "module", Short: "Manage policy group modules"}
	cmd.AddCommand(newPolicyGroupModuleSetCommand())
	return cmd
}

func newPolicyGroupModuleSetCommand() *cobra.Command {
	var modules []string
	var state string
	var opts writeOptions
	cmd := &cobra.Command{Use: "set <id>", Short: "Set policy group module state", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if len(splitValues(modules)) == 0 || state == "" {
			return fmt.Errorf("--module and --state are required")
		}
		current, err := fetchPolicyGroup(cmd, id)
		if err != nil {
			return err
		}
		modConfig, ok := current["modules_detection_config"].(map[string]any)
		if !ok {
			return fmt.Errorf("policy group %d has invalid modules_detection_config", id)
		}
		modulesMap, ok := modConfig["modules"].(map[string]any)
		if !ok {
			return fmt.Errorf("policy group %d has invalid modules_detection_config.modules", id)
		}
		for _, module := range splitValues(modules) {
			item, ok := modulesMap[module].(map[string]any)
			if !ok {
				return fmt.Errorf("module %q not found in policy group %d", module, id)
			}
			item["state"] = state
		}
		body := map[string]any{
			"id":                            id,
			"name":                          stringField(current, "name"),
			"comment":                       stringField(current, "comment"),
			"modules_detection_config":      modConfig,
			"base_modules_detection_config": current["base_modules_detection_config"],
		}
		return doWrite(cmd, opts, "policy-group.module.set", http.MethodPut, "/api/v3/detect/PolicyGroup", nil, body, []string{"SafeLine-3 policy-group update requires full module config; CLI reads current detail and updates selected module states"})
	}}
	cmd.Flags().StringArrayVar(&modules, "module", nil, "Module name; repeatable or comma separated")
	cmd.Flags().StringVar(&state, "state", "", "enabled|disabled")
	addWriteFlags(cmd, &opts)
	return cmd
}

func fetchPolicyGroup(cmd *cobra.Command, id uint64) (map[string]any, error) {
	var item map[string]any
	q := url.Values{"id": {strconv.FormatUint(id, 10)}}
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/detect/PolicyGroup/detail", q, nil, &item); err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("policy group %d not found", id)
	}
	return item, nil
}

func normalizePolicyRuleAction(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "deny", "block", "blocked", "forbid", "forbidden":
		return "deny", nil
	case "allow", "pass", "accept":
		return "allow", nil
	case "dry-run", "dryrun", "dry_run":
		return "dry_run", nil
	case "modify-module", "modify_module":
		return "modify_module", nil
	case "modify-skynet-rule", "modify_skynet_rule":
		return "modify_skynet_rule", nil
	default:
		return "", fmt.Errorf("invalid action %q (expected deny|allow|dry-run|modify-module|modify-skynet-rule)", value)
	}
}

func parsePolicyRuleActionCode(value string) (int64, error) {
	if code, err := strconv.ParseInt(value, 10, 32); err == nil {
		return code, nil
	}
	action, err := normalizePolicyRuleAction(value)
	if err != nil {
		return 0, err
	}
	switch action {
	case "allow":
		return 0, nil
	case "deny":
		return 1, nil
	case "dry_run":
		return 2, nil
	case "modify_module":
		return 3, nil
	case "modify_skynet_rule":
		return 5, nil
	default:
		return 0, fmt.Errorf("invalid action %q", value)
	}
}

func simplePolicyRuleCondition(target, op string, values []string) map[string]any {
	return map[string]any{
		"set_not":                   false,
		"operator":                  normalizePolicyRuleOperator(op),
		"match_key":                 normalizePolicyRuleMatchKey(target),
		"values":                    values,
		"children":                  []any{},
		"decode_methods":            []any{},
		"custom_conflicts_group_id": 0,
	}
}

func normalizePolicyRuleMatchKey(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "src_ip", "source_ip", "remote_ip", "remote_addr":
		return "remote_addr"
	case "url", "path", "url_path", "urlpath":
		return "urlpath"
	case "header", "request_header":
		return "request_header"
	default:
		return value
	}
}

func normalizePolicyRuleOperator(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "infix", "contains", "contain":
		return "infix"
	case "str", "eq", "equals", "equal":
		return "str"
	case "regex", "regexp", "re":
		return "re"
	case "in", "cidr":
		return "cidr"
	default:
		return value
	}
}

func normalizePolicyLogOption(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "persistence", "persistent":
		return "Persistence", nil
	case "non-persistence", "nonpersistent", "non-persistent":
		return "Non-Persistence", nil
	case "drop":
		return "Drop", nil
	default:
		return "", fmt.Errorf("invalid log option %q", value)
	}
}

func parsePolicyRiskLevel(value string) (int32, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return -1, nil
	default:
		return parseRiskLevel(value)
	}
}

func newPolicyGroupDeleteCommand() *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{Use: "delete <id...>", Short: "Delete policy groups", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		return doWrite(cmd, opts, "policy-group.delete", http.MethodDelete, "/api/v3/detect/PolicyGroup", nil, map[string]any{"ids": ids}, nil)
	}}
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "policy-rule", Short: "Manage custom policy rules"}
	cmd.AddCommand(newPolicyRuleListCommand())
	cmd.AddCommand(newPolicyRuleGetCommand())
	cmd.AddCommand(newPolicyRuleCreateCommand())
	cmd.AddCommand(newPolicyRuleUpdateCommand())
	cmd.AddCommand(newPolicyRuleEnableCommand("enable", true))
	cmd.AddCommand(newPolicyRuleEnableCommand("disable", false))
	cmd.AddCommand(newPolicyRuleDeleteCommand())
	cmd.AddCommand(newPolicyRuleMoveCommand())
	cmd.AddCommand(newPolicyRuleBindingCommand("bind", http.MethodPost))
	cmd.AddCommand(newPolicyRuleBindingCommand("unbind", http.MethodDelete))
	return cmd
}

func newPolicyRuleListCommand() *cobra.Command {
	var global bool
	var appID uint64
	var name, enabled, action, risk string
	var page, pageSize int
	cmd := &cobra.Command{Use: "list", Short: "List policy rules", RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{"offset": (page - 1) * pageSize, "count": pageSize, "is_global": valueCondition(global)}
		if appID != 0 {
			body["app_ids"] = valueCondition([]uint64{appID})
		}
		if name != "" {
			body["name"] = equalCondition(name)
		}
		if enabled != "" {
			body["status"] = equalCondition(enabled)
		}
		if action != "" {
			value, err := parsePolicyRuleActionCode(action)
			if err != nil {
				return err
			}
			body["action"] = equalCondition(value)
		}
		if risk != "" {
			value, err := parseRiskLevel(risk)
			if err != nil {
				return err
			}
			body["risk_level"] = equalCondition(value)
		}
		return doRequest(cmd, http.MethodPost, "/api/v3/detect/PolicyRule/filter", nil, body)
	}}
	cmd.Flags().BoolVar(&global, "global", true, "Query global rules")
	cmd.Flags().Uint64Var(&appID, "app-id", 0, "Application ID")
	cmd.Flags().StringVar(&name, "name", "", "Name filter")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Enabled true|false")
	cmd.Flags().StringVar(&action, "action", "", "Action filter")
	cmd.Flags().StringVar(&risk, "risk-level", "", "Risk level")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newPolicyRuleGetCommand() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get policy rule by filtering ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		return doRequest(cmd, http.MethodPost, "/api/v3/detect/PolicyRule/filter", nil, map[string]any{"id": equalCondition(id), "is_global": valueCondition(true), "offset": 0, "count": 1})
	}}
}

func newPolicyRuleCreateSimpleCommand() *cobra.Command {
	var name, target, op, action, risk, bind string
	var values []string
	var appID uint64
	var enabled bool
	var logOption string
	var opts writeOptions
	cmd := &cobra.Command{Use: "simple", Short: "Create a simple policy rule", RunE: func(cmd *cobra.Command, args []string) error {
		if name == "" || target == "" || op == "" || action == "" || len(splitValues(values)) == 0 {
			return fmt.Errorf("--name, --target, --op, --value and --action are required")
		}
		if bind == "app" && appID == 0 {
			return fmt.Errorf("--app-id is required when --bind app")
		}
		riskLevel, err := parsePolicyRiskLevel(risk)
		if err != nil {
			return err
		}
		action, err = normalizePolicyRuleAction(action)
		if err != nil {
			return err
		}
		logOption, err = normalizePolicyLogOption(logOption)
		if err != nil {
			return err
		}
		bindingType := "Global"
		if bind == "app" {
			bindingType = "AppBinding"
		}
		body := map[string]any{
			"name":          name,
			"condition":     simplePolicyRuleCondition(target, op, splitValues(values)),
			"action":        action,
			"attack_type":   -1,
			"risk_level":    riskLevel,
			"is_enabled":    enabled,
			"log_option":    logOption,
			"schedule_type": "all",
			"binding":       []map[string]any{{"binding_type": bindingType, "id": appID}},
		}
		return doWrite(cmd, opts, "policy-rule.create.simple", http.MethodPost, "/api/v3/detect/PolicyRule", nil, body, []string{"simple mode builds a compact condition; use policy-rule create --condition-file for exact API condition JSON"})
	}}
	cmd.Flags().StringVar(&name, "name", "", "Rule name")
	cmd.Flags().StringVar(&target, "target", "", "Match target")
	cmd.Flags().StringVar(&op, "op", "", "Match operator")
	cmd.Flags().StringArrayVar(&values, "value", nil, "Match value")
	cmd.Flags().StringVar(&action, "action", "", "Action")
	cmd.Flags().StringVar(&risk, "risk-level", "none", "Risk level none|low|medium|high|0|1|2|3")
	cmd.Flags().StringVar(&bind, "bind", "global", "Binding scope global|app")
	cmd.Flags().Uint64Var(&appID, "app-id", 0, "Application ID")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable rule")
	cmd.Flags().StringVar(&logOption, "log-option", "persistence", "Log option persistence|non-persistence|drop")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleCreateCommand() *cobra.Command {
	var name, conditionFile, bindingFile, scheduleType, scheduleFile, action string
	var opts writeOptions
	cmd := &cobra.Command{Use: "create", Short: "Create policy rule from files", RunE: func(cmd *cobra.Command, args []string) error {
		if name == "" || conditionFile == "" || action == "" {
			return fmt.Errorf("--name, --condition-file and --action are required")
		}
		condition, err := readJSONFile(conditionFile)
		if err != nil {
			return err
		}
		action, err = normalizePolicyRuleAction(action)
		if err != nil {
			return err
		}
		body := map[string]any{
			"is_enabled":      true,
			"name":            name,
			"comment":         "",
			"condition":       condition,
			"binding":         []map[string]any{{"binding_type": "Global", "id": 0}},
			"action":          action,
			"rule_type":       "detect_rule",
			"log_option":      "Persistence",
			"schedule_type":   "all",
			"schedule_config": map[string]any{},
			"attack_type":     0,
			"risk_level":      0,
		}
		if bindingFile != "" {
			body["binding"], err = readJSONFile(bindingFile)
			if err != nil {
				return err
			}
		}
		if scheduleType != "" {
			body["schedule_type"] = scheduleType
		}
		if scheduleFile != "" {
			body["schedule_config"], err = readJSONFile(scheduleFile)
			if err != nil {
				return err
			}
		}
		return doWrite(cmd, opts, "policy-rule.create", http.MethodPost, "/api/v3/detect/PolicyRule", nil, body, nil)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Rule name")
	cmd.Flags().StringVar(&conditionFile, "condition-file", "", "Condition JSON file or -")
	cmd.Flags().StringVar(&bindingFile, "binding-file", "", "Binding JSON file or -")
	cmd.Flags().StringVar(&scheduleType, "schedule-type", "", "Schedule type")
	cmd.Flags().StringVar(&scheduleFile, "schedule-file", "", "Schedule config JSON file or -")
	cmd.Flags().StringVar(&action, "action", "", "Action")
	addWriteFlags(cmd, &opts)
	cmd.AddCommand(newPolicyRuleCreateSimpleCommand())
	return cmd
}

func newPolicyRuleUpdateCommand() *cobra.Command {
	var payload string
	var opts writeOptions
	cmd := &cobra.Command{Use: "update <id>", Short: "Update policy rule from payload file", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
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
		return doWrite(cmd, opts, "policy-rule.update", http.MethodPut, "/api/v3/detect/PolicyRule", nil, body, nil)
	}}
	cmd.Flags().StringVar(&payload, "payload-file", "", "Full policy rule payload JSON file or -")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleEnableCommand(name string, enabled bool) *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{Use: name + " <id...>", Short: name + " policy rules", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		return doWrite(cmd, opts, "policy-rule."+name, http.MethodPut, "/api/v3/detect/PolicyRule/set_enabled", nil, map[string]any{"ids": ids, "is_enabled": enabled}, nil)
	}}
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleDeleteCommand() *cobra.Command {
	var opts writeOptions
	cmd := &cobra.Command{Use: "delete <id...>", Short: "Delete policy rules", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		return doWrite(cmd, opts, "policy-rule.delete", http.MethodDelete, "/api/v3/detect/PolicyRule", nil, map[string]any{"ids": ids}, nil)
	}}
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleMoveCommand() *cobra.Command {
	var position int
	var global bool
	var appID uint64
	var opts writeOptions
	cmd := &cobra.Command{Use: "move <id>", Short: "Move policy rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if position <= 0 {
			return fmt.Errorf("--position must be >= 1")
		}
		body := map[string]any{"id": id, "position": position, "is_global": global, "app_id": appID}
		return doWrite(cmd, opts, "policy-rule.move", http.MethodPut, "/api/v3/detect/PolicyRule/priority", nil, body, nil)
	}}
	cmd.Flags().IntVar(&position, "position", 0, "Target position")
	cmd.Flags().BoolVar(&global, "global", true, "Global rules")
	cmd.Flags().Uint64Var(&appID, "app-id", 0, "Application ID")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newPolicyRuleBindingCommand(name, method string) *cobra.Command {
	var appID uint64
	var opts writeOptions
	cmd := &cobra.Command{Use: name + " <id...>", Short: name + " policy rules to an application", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		if appID == 0 {
			return fmt.Errorf("--app-id is required")
		}
		body := map[string]any{"application_id": appID, "policy_rule_ids": ids}
		return doWrite(cmd, opts, "policy-rule."+name, method, "/api/v3/detect/PolicyRule/binding", nil, body, nil)
	}}
	cmd.Flags().Uint64Var(&appID, "app-id", 0, "Application ID")
	addWriteFlags(cmd, &opts)
	return cmd
}
