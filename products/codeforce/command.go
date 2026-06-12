package codeforce

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codeforce",
		Short: "CodeForce project, repository, AI task, and denoise management",
		Long: `CodeForce CLI

Manage CodeForce projects, project AI employees, AI development tasks, native
audit tasks, denoise tasks, code packages, project repositories, and personal
Git authorization configs from chaitin-cli.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			applyRuntimeConfig(cmd)
		},
	}

	cmd.PersistentFlags().String("url", "", "CodeForce URL")
	cmd.PersistentFlags().String("access-token", "", "CodeForce access token")
	cmd.PersistentFlags().String("api-key", "", "Alias for --access-token")
	cmd.PersistentFlags().String("account-type", accountTypeAdmin, "Account type (admin|user|openapi)")
	cmd.PersistentFlags().Bool("insecure", true, "Skip TLS certificate verification")
	cmd.PersistentFlags().Int("timeout", defaultTimeoutSeconds, "HTTP request timeout in seconds")

	cmd.AddCommand(newProjectCommand())
	cmd.AddCommand(newAuditCommand())
	cmd.AddCommand(newDenoiseCommand())
	cmd.AddCommand(newCodeManagementCommand())
	cmd.AddCommand(newRepositoryCommand())
	cmd.AddCommand(newGitAuthCommand())
	cmd.AddCommand(newOpenAPICommand())
	return cmd
}

func newProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage CodeForce projects",
	}
	cmd.AddCommand(newProjectCreateCommand())
	cmd.AddCommand(newProjectAIEmployeeCommand())
	cmd.AddCommand(newProjectAIDevCommand())
	return cmd
}

func newProjectCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a CodeForce project",
		RunE:  runProjectCreate,
	}
	cmd.Flags().String("name", "", "Project name")
	cmd.Flags().String("description", "", "Project description")
	cmd.Flags().String("repository-id", "", "Repository ID bound to the project")
	return cmd
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "project create"); err != nil {
		return err
	}
	name := mustGetString(cmd, "name")
	repositoryID := mustGetString(cmd, "repository-id")
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if repositoryID == "" {
		return fmt.Errorf("please provide --repository-id")
	}
	body := map[string]any{
		"name":          name,
		"description":   mustGetString(cmd, "description"),
		"repository_id": repositoryID,
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, "/api/v1/codeforce/projects", nil, body, nil),
		})
	}
	data, err := NewClient(cfg).projectCreate(cmd.Context(), body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"project_id":         valueString(data["id"]),
		"project_name":       firstNonEmpty(valueString(data["name"]), name),
		"repository_id":      firstNonEmpty(valueString(data["repository_id"]), repositoryID),
		"current_user_role":  valueString(data["current_user_role"]),
		"project":            data,
		"management_account": cfg.AccountType,
	})
}

func newProjectAIEmployeeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-employee",
		Short: "Manage project AI employees",
	}
	cmd.AddCommand(newProjectAIEmployeeModelOptionsCommand())
	cmd.AddCommand(newProjectAIEmployeeCreateCommand())
	return cmd
}

func newProjectAIEmployeeModelOptionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model-options",
		Short: "List project AI employee model options",
		RunE:  runProjectAIEmployeeModelOptions,
	}
	cmd.Flags().String("project-id", "", "Project ID")
	cmd.Flags().String("employee-id", "", "Existing AI employee ID")
	return cmd
}

func runProjectAIEmployeeModelOptions(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "project ai-employee model-options"); err != nil {
		return err
	}
	projectID := mustGetString(cmd, "project-id")
	if projectID == "" {
		return fmt.Errorf("please provide --project-id")
	}
	data, err := NewClient(cfg).projectAIEmployeeModelOptions(cmd.Context(), projectID, mustGetString(cmd, "employee-id"))
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"project_id": projectID,
		"models":     data,
	})
}

func newProjectAIEmployeeCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project AI employee",
		RunE:  runProjectAIEmployeeCreate,
	}
	cmd.Flags().String("project-id", "", "Project ID")
	cmd.Flags().String("type", "", "AI employee type (dev|security)")
	cmd.Flags().String("name", "", "AI employee name")
	cmd.Flags().String("primary-model-id", "", "Primary model ID")
	cmd.Flags().String("backup-model-id", "", "Backup model ID")
	cmd.Flags().Bool("enabled", false, "Enable the AI employee immediately")
	cmd.Flags().String("trigger-config-json", "", "Trigger config JSON object")
	cmd.Flags().String("prompt-config-json", "", "Prompt config JSON object")
	cmd.Flags().StringSlice("audit-template-id", nil, "Audit template ID, repeatable")
	return cmd
}

func runProjectAIEmployeeCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "project ai-employee create"); err != nil {
		return err
	}
	projectID := mustGetString(cmd, "project-id")
	employeeType := strings.ToLower(mustGetString(cmd, "type"))
	name := mustGetString(cmd, "name")
	if projectID == "" {
		return fmt.Errorf("please provide --project-id")
	}
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if employeeType != "dev" && employeeType != "security" {
		return fmt.Errorf("please provide --type dev or --type security")
	}
	triggerConfig, err := parseStringMapJSON(mustGetString(cmd, "trigger-config-json"), "trigger-config-json")
	if err != nil {
		return err
	}
	promptConfig, err := parseStringMapJSON(mustGetString(cmd, "prompt-config-json"), "prompt-config-json")
	if err != nil {
		return err
	}
	auditTemplateIDs, _ := cmd.Flags().GetStringSlice("audit-template-id")
	body := map[string]any{
		"type":               employeeType,
		"name":               name,
		"enabled":            mustGetBool(cmd, "enabled"),
		"trigger_config":     triggerConfig,
		"prompt_config":      promptConfig,
		"audit_template_ids": auditTemplateIDs,
	}
	if modelID := mustGetString(cmd, "primary-model-id"); modelID != "" {
		body["primary_model_id"] = modelID
	}
	if modelID := mustGetString(cmd, "backup-model-id"); modelID != "" {
		body["backup_model_id"] = modelID
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, "/api/v1/codeforce/projects/"+projectID+"/ai-employees", nil, body, nil),
		})
	}
	data, err := NewClient(cfg).projectAIEmployeeCreate(cmd.Context(), projectID, body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"project_id":          projectID,
		"employee_id":         valueString(data["id"]),
		"employee_type":       firstNonEmpty(valueString(data["type"]), employeeType),
		"employee_name":       firstNonEmpty(valueString(data["name"]), name),
		"enabled":             data["enabled"],
		"webhook_sync_status": valueString(data["webhook_sync_status"]),
		"webhook_results":     data["webhook_results"],
		"project_ai_employee": data,
		"management_account":  cfg.AccountType,
	})
}

func newProjectAIDevCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-dev",
		Short: "Manage project AI development tasks",
	}
	cmd.AddCommand(newProjectAIDevCreateCommand())
	cmd.AddCommand(newProjectAIDevResultCommand())
	return cmd
}

func newProjectAIDevCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project AI development task",
		RunE:  runProjectAIDevCreate,
	}
	cmd.Flags().String("project-id", "", "Project ID")
	cmd.Flags().String("employee-id", "", "AI employee ID")
	cmd.Flags().String("title", "", "Task title")
	cmd.Flags().String("issue-url", "", "Issue URL")
	cmd.Flags().String("branch", "", "Target branch name")
	return cmd
}

func runProjectAIDevCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "project ai-dev create"); err != nil {
		return err
	}
	projectID := mustGetString(cmd, "project-id")
	employeeID := mustGetString(cmd, "employee-id")
	title := mustGetString(cmd, "title")
	branch := mustGetString(cmd, "branch")
	if projectID == "" {
		return fmt.Errorf("please provide --project-id")
	}
	if employeeID == "" {
		return fmt.Errorf("please provide --employee-id")
	}
	if title == "" {
		return fmt.Errorf("please provide --title")
	}
	if branch == "" {
		return fmt.Errorf("please provide --branch")
	}
	body := map[string]any{
		"employee_id": employeeID,
		"title":       title,
		"issue_url":   mustGetString(cmd, "issue-url"),
		"branch":      branch,
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, "/api/v1/codeforce/projects/"+projectID+"/ai-dev/tasks", nil, body, nil),
		})
	}
	data, err := NewClient(cfg).projectAIDevCreate(cmd.Context(), projectID, body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"project_id":         firstNonEmpty(valueString(data["project_id"]), projectID),
		"task_id":            valueString(data["id"]),
		"employee_id":        firstNonEmpty(valueString(data["employee_id"]), employeeID),
		"title":              firstNonEmpty(valueString(data["title"]), title),
		"branch":             firstNonEmpty(valueString(data["branch"]), branch),
		"status":             valueString(data["status"]),
		"project_ai_task":    data,
		"management_account": cfg.AccountType,
	})
}

func newProjectAIDevResultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "result",
		Short: "Get a project AI development task result",
		RunE:  runProjectAIDevResult,
	}
	cmd.Flags().String("project-id", "", "Project ID")
	cmd.Flags().String("task-id", "", "AI dev task ID")
	return cmd
}

func runProjectAIDevResult(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "project ai-dev result"); err != nil {
		return err
	}
	projectID := mustGetString(cmd, "project-id")
	taskID := mustGetString(cmd, "task-id")
	if projectID == "" {
		return fmt.Errorf("please provide --project-id")
	}
	if taskID == "" {
		return fmt.Errorf("please provide --task-id")
	}
	data, err := NewClient(cfg).projectAIDevResult(cmd.Context(), projectID, taskID)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"project_id":      projectID,
		"task_id":         firstNonEmpty(valueString(data["id"]), taskID),
		"status":          valueString(data["status"]),
		"summary":         data["summary"],
		"result_payload":  data["result_payload"],
		"task_logs":       data["task_logs"],
		"error_logs":      data["error_logs"],
		"error_message":   data["error_message"],
		"project_ai_task": data,
	})
}

func newAuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Manage CodeForce audit tasks",
	}
	cmd.AddCommand(newAuditNativeCommand())
	return cmd
}

func newAuditNativeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "native",
		Short: "Manage native audit tasks",
	}
	cmd.AddCommand(newAuditNativeCreateCommand())
	cmd.AddCommand(newAuditNativeResultCommand())
	cmd.AddCommand(newAuditNativeExportOverviewCommand())
	return cmd
}

func newAuditNativeCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create native audit tasks",
	}
	cmd.AddCommand(newAuditNativeCreateGitCommand())
	cmd.AddCommand(newAuditNativeCreateCodeCommand())
	return cmd
}

func newAuditNativeCreateGitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Create a native audit task in Git repository mode",
		RunE:  runAuditNativeCreateGit,
	}
	cmd.Flags().String("repository-id", "", "Repository ID")
	cmd.Flags().String("source-ref", "", "Source ref, for example branch:main")
	cmd.Flags().String("target-ref", "", "Target ref, for example branch:release/2026.06")
	cmd.Flags().StringSlice("audit-rule-id", nil, "Audit rule ID, repeatable")
	cmd.Flags().String("task-name", "", "Task name")
	return cmd
}

func runAuditNativeCreateGit(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "audit native create git"); err != nil {
		return err
	}
	repositoryID := mustGetString(cmd, "repository-id")
	if repositoryID == "" {
		return fmt.Errorf("please provide --repository-id")
	}
	sourceRef, err := parseRefSpec(mustGetString(cmd, "source-ref"), "source-ref", true)
	if err != nil {
		return err
	}
	targetRef, err := parseRefSpec(mustGetString(cmd, "target-ref"), "target-ref", false)
	if err != nil {
		return err
	}
	auditRuleIDs, _ := cmd.Flags().GetStringSlice("audit-rule-id")
	if len(auditRuleIDs) == 0 {
		return fmt.Errorf("please provide at least one --audit-rule-id")
	}
	body := map[string]any{
		"repository_id":  repositoryID,
		"source_ref":     sourceRef,
		"audit_rule_ids": auditRuleIDs,
		"task_name":      mustGetString(cmd, "task-name"),
		"repo_mode":      "git",
	}
	if targetRef != nil {
		body["target_ref"] = targetRef
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, managementPathForDryRun(cfg, "/api/v1/aiemployee/aitask/manual", "/api/v1/user/aiemployee/aitask/manual"), nil, body, nil),
		})
	}
	data, err := NewClient(cfg).nativeAuditCreate(cmd.Context(), body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"task_ids":            data["task_ids"],
		"repository_id":       repositoryID,
		"audit_rule_ids":      auditRuleIDs,
		"repo_mode":           "git",
		"native_audit_result": data,
	})
}

func newAuditNativeCreateCodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code",
		Short: "Create a native audit task in code package mode",
		RunE:  runAuditNativeCreateCode,
	}
	cmd.Flags().String("code-repository-id", "", "Code repository ID")
	cmd.Flags().String("code-repository-name", "", "Code repository name")
	cmd.Flags().String("head-file-path", "", "Head version file path")
	cmd.Flags().String("base-file-path", "", "Base version file path")
	cmd.Flags().String("head-version-label", "", "Head version label")
	cmd.Flags().String("base-version-label", "", "Base version label")
	cmd.Flags().StringSlice("audit-rule-id", nil, "Audit rule ID, repeatable")
	cmd.Flags().String("task-name", "", "Task name")
	return cmd
}

func runAuditNativeCreateCode(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "audit native create code"); err != nil {
		return err
	}
	codeRepositoryID := mustGetString(cmd, "code-repository-id")
	headFilePath := mustGetString(cmd, "head-file-path")
	if codeRepositoryID == "" {
		return fmt.Errorf("please provide --code-repository-id")
	}
	if headFilePath == "" {
		return fmt.Errorf("please provide --head-file-path")
	}
	auditRuleIDs, _ := cmd.Flags().GetStringSlice("audit-rule-id")
	if len(auditRuleIDs) == 0 {
		return fmt.Errorf("please provide at least one --audit-rule-id")
	}
	body := map[string]any{
		"repo_mode":            "code",
		"code_repository_id":   codeRepositoryID,
		"code_repository_name": mustGetString(cmd, "code-repository-name"),
		"head_file_path":       headFilePath,
		"base_file_path":       mustGetString(cmd, "base-file-path"),
		"head_version_label":   mustGetString(cmd, "head-version-label"),
		"base_version_label":   mustGetString(cmd, "base-version-label"),
		"audit_rule_ids":       auditRuleIDs,
		"task_name":            mustGetString(cmd, "task-name"),
		"source_ref": map[string]any{
			"type": "branch",
			"name": "code-mode-placeholder",
		},
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, managementPathForDryRun(cfg, "/api/v1/aiemployee/aitask/manual", "/api/v1/user/aiemployee/aitask/manual"), nil, body, nil),
		})
	}
	data, err := NewClient(cfg).nativeAuditCreate(cmd.Context(), body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"task_ids":            data["task_ids"],
		"code_repository_id":  codeRepositoryID,
		"repo_mode":           "code",
		"native_audit_result": data,
	})
}

func newAuditNativeResultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "result",
		Short: "Get a native audit task result",
		RunE:  runAuditNativeResult,
	}
	cmd.Flags().String("task-id", "", "Task ID")
	return cmd
}

func runAuditNativeResult(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	taskID := mustGetString(cmd, "task-id")
	if taskID == "" {
		return fmt.Errorf("please provide --task-id")
	}
	client := NewClient(cfg)
	var data map[string]any
	switch cfg.AccountType {
	case accountTypeOpenAPI:
		data, err = client.openAPIManualAIAuditTaskStatus(cmd.Context(), taskID)
	default:
		data, err = client.nativeAuditResult(cmd.Context(), taskID)
	}
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"task_id":             taskID,
		"account_type":        cfg.AccountType,
		"native_audit_result": data,
		"status":              firstNonEmpty(valueString(data["status"]), valueString(data["task_status"])),
		"summary":             firstNonEmpty(valueString(data["summary"]), valueString(data["task_summary"])),
		"error_message":       firstNonEmpty(valueString(data["error_message"]), valueString(data["task_error"])),
	})
}

func newAuditNativeExportOverviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-overview",
		Short: "Export native audit manual overview as CSV",
		RunE:  runAuditNativeExportOverview,
	}
	cmd.Flags().String("keyword", "", "Keyword filter")
	cmd.Flags().String("status", "", "Status filter")
	cmd.Flags().String("repository-name", "", "Repository name filter")
	cmd.Flags().String("creator-name", "", "Creator name filter")
	cmd.Flags().String("out", "", "Output file path")
	return cmd
}

func runAuditNativeExportOverview(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "audit native export-overview"); err != nil {
		return err
	}
	query := url.Values{}
	for _, item := range []struct {
		flag string
		key  string
	}{
		{flag: "keyword", key: "keyword"},
		{flag: "status", key: "status"},
		{flag: "repository-name", key: "repository_name"},
		{flag: "creator-name", key: "creator_name"},
	} {
		if value := mustGetString(cmd, item.flag); value != "" {
			query.Set(item.key, value)
		}
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodGet, managementPathForDryRun(cfg, "/api/v1/aiemployee/aitask/manual-overview/export", "/api/v1/user/aiemployee/aitask/manual-overview/export"), rawQuery(query), nil, nil),
		})
	}
	data, headers, err := NewClient(cfg).manualOverviewExport(cmd.Context(), query)
	if err != nil {
		return err
	}
	out := mustGetString(cmd, "out")
	if out == "" {
		out = filenameFromHeader(headers, "manual_audit_overview.csv")
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := writeOutputFile(out, data); err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"out":   out,
		"bytes": len(data),
	})
}

func newDenoiseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoise",
		Short: "Manage CodeForce denoise tasks",
	}
	cmd.AddCommand(newDenoiseParseCommand())
	cmd.AddCommand(newDenoiseCreateCommand())
	cmd.AddCommand(newDenoiseResultCommand())
	return cmd
}

func newDenoiseParseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse a denoise report without creating a task",
		RunE:  runDenoiseParse,
	}
	cmd.Flags().String("type", "sast", "Report type (sast|sca)")
	cmd.Flags().String("report-file", "", "Report file path")
	return cmd
}

func runDenoiseParse(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	taskType, err := normalizeDenoiseType(mustGetString(cmd, "type"))
	if err != nil {
		return err
	}
	reportFile := mustGetString(cmd, "report-file")
	if reportFile == "" {
		return fmt.Errorf("please provide --report-file")
	}
	if err := requireReadableFile(reportFile, "report-file"); err != nil {
		return err
	}
	client := NewClient(cfg)
	if dryRun {
		path := client.managementPath("/api/v1/aiemployee/denoise/tasks/parse", "/api/v1/user/aiemployee/denoise/tasks/parse")
		if cfg.AccountType == accountTypeOpenAPI {
			path = "/api/v1/codeforce/openapi/denoise-tasks/parse"
		}
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, path, nil, map[string]any{"type": taskType}, map[string]string{"file": reportFile}),
		})
	}
	var result *reportParseResult
	switch cfg.AccountType {
	case accountTypeOpenAPI:
		result, err = client.openAPIParseDenoiseReport(cmd.Context(), map[string]string{"type": taskType}, map[string]string{"file": reportFile})
	default:
		result, err = client.denoiseParse(cmd.Context(), taskType, reportFile)
	}
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"type":                                  result.Type,
		"max_selected_vulnerabilities_per_task": result.MaxSelectedVulnerabilitiesPerTask,
		"max_minio_zip_file_size_mb":            result.MaxMinioZipFileSizeMB,
		"vulnerabilities":                       result.Vulnerabilities,
		"sca_vulnerabilities":                   result.ScaVulnerabilities,
	})
}

func newDenoiseCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a CodeForce denoise task",
		RunE:  runDenoiseCreate,
	}
	cmd.Flags().String("type", "sast", "Task type (sast|sca)")
	cmd.Flags().String("name", "", "Task name")
	cmd.Flags().String("engineer-id", "", "Denoise engineer ID")
	cmd.Flags().String("engineer-name", "", "Denoise engineer name for OpenAPI mode")
	cmd.Flags().String("source-type", "", "Source type (repository|zip)")
	cmd.Flags().String("repository-id", "", "Repository ID")
	cmd.Flags().String("repository-name", "", "Repository name, used directly in OpenAPI mode")
	cmd.Flags().String("branch-or-tag", "", "Repository branch or tag")
	cmd.Flags().String("zip-file", "", "ZIP file path for zip source mode")
	cmd.Flags().String("report-file", "", "SAST/SCA report file path")
	cmd.Flags().String("selection-json", "", "OpenAPI selection JSON, defaults to {\"mode\":\"all\"}")
	return cmd
}

func runDenoiseCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	taskType, err := normalizeDenoiseType(mustGetString(cmd, "type"))
	if err != nil {
		return err
	}
	sourceType := strings.ToLower(mustGetString(cmd, "source-type"))
	name := mustGetString(cmd, "name")
	engineerID := mustGetString(cmd, "engineer-id")
	reportFile := mustGetString(cmd, "report-file")
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if reportFile == "" {
		return fmt.Errorf("please provide --report-file")
	}
	if err := requireReadableFile(reportFile, "report-file"); err != nil {
		return err
	}
	if sourceType != "repository" && sourceType != "zip" {
		return fmt.Errorf("please provide --source-type repository or --source-type zip")
	}

	client := NewClient(cfg)
	switch cfg.AccountType {
	case accountTypeOpenAPI:
		return runDenoiseCreateOpenAPI(cmd, client, cfg, taskType, sourceType, name, engineerID, reportFile)
	default:
		if engineerID == "" {
			return fmt.Errorf("please provide --engineer-id")
		}
		return runDenoiseCreateManagement(cmd, client, cfg, taskType, sourceType, name, engineerID, reportFile)
	}
}

func runDenoiseCreateManagement(cmd *cobra.Command, client *Client, cfg Config, taskType, sourceType, name, engineerID, reportFile string) error {
	fields := map[string]string{
		"name":        name,
		"engineer_id": engineerID,
		"type":        taskType,
		"source_type": sourceType,
	}
	if sourceType == "repository" {
		repositoryID := mustGetString(cmd, "repository-id")
		branchOrTag := mustGetString(cmd, "branch-or-tag")
		if repositoryID == "" {
			return fmt.Errorf("please provide --repository-id")
		}
		if branchOrTag == "" {
			return fmt.Errorf("please provide --branch-or-tag")
		}
		fields["repository_id"] = repositoryID
		fields["branch_or_tag"] = branchOrTag
	}
	if sourceType == "zip" {
		zipFile := mustGetString(cmd, "zip-file")
		if zipFile == "" {
			return fmt.Errorf("please provide --zip-file")
		}
		if err := requireReadableFile(zipFile, "zip-file"); err != nil {
			return err
		}
		if dryRun {
			requests := []dryRunRequest{
				makeDryRunRequest(cfg, http.MethodPost, client.managementPath("/api/v1/aiemployee/denoise/tasks/parse", "/api/v1/user/aiemployee/denoise/tasks/parse"), nil, map[string]any{"type": taskType}, map[string]string{"file": reportFile}),
				makeDryRunRequest(cfg, http.MethodPost, client.managementPath("/api/v1/aiemployee/denoise/tasks", "/api/v1/user/aiemployee/denoise/tasks"), nil, fields, map[string]string{
					"zip_file": zipFile,
				}),
			}
			return outputDryRun(cmd, requests)
		}
		parseResult, err := client.denoiseParse(cmd.Context(), taskType, reportFile)
		if err != nil {
			return err
		}
		payloadFields := cloneStringMap(fields)
		var key string
		var raw string
		if taskType == "sca" {
			key = "sca_vulnerabilities"
			raw, err = mustJSONEncode(parseResult.ScaVulnerabilities)
		} else {
			key = "vulnerabilities"
			raw, err = mustJSONEncode(parseResult.Vulnerabilities)
		}
		if err != nil {
			return fmt.Errorf("encode parsed vulnerabilities: %w", err)
		}
		payloadFields[key] = raw
		data, err := client.denoiseCreate(cmd.Context(), payloadFields, map[string]string{"zip_file": zipFile}, true)
		if err != nil {
			return err
		}
		return outputDenoiseCreate(cmd, cfg, data, name, sourceType, taskType, engineerID)
	}

	if dryRun {
		requests := []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, client.managementPath("/api/v1/aiemployee/denoise/tasks/parse", "/api/v1/user/aiemployee/denoise/tasks/parse"), nil, map[string]any{"type": taskType}, map[string]string{"file": reportFile}),
			makeDryRunRequest(cfg, http.MethodPost, client.managementPath("/api/v1/aiemployee/denoise/tasks", "/api/v1/user/aiemployee/denoise/tasks"), nil, fields, nil),
		}
		return outputDryRun(cmd, requests)
	}
	parseResult, err := client.denoiseParse(cmd.Context(), taskType, reportFile)
	if err != nil {
		return err
	}
	if taskType == "sca" {
		raw, err := mustJSONEncode(parseResult.ScaVulnerabilities)
		if err != nil {
			return fmt.Errorf("encode parsed vulnerabilities: %w", err)
		}
		fields["sca_vulnerabilities"] = raw
	} else {
		raw, err := mustJSONEncode(parseResult.Vulnerabilities)
		if err != nil {
			return fmt.Errorf("encode parsed vulnerabilities: %w", err)
		}
		fields["vulnerabilities"] = raw
	}
	data, err := client.denoiseCreate(cmd.Context(), fields, nil, false)
	if err != nil {
		return err
	}
	return outputDenoiseCreate(cmd, cfg, data, name, sourceType, taskType, engineerID)
}

func runDenoiseCreateOpenAPI(cmd *cobra.Command, client *Client, cfg Config, taskType, sourceType, name, engineerID, reportFile string) error {
	fields := map[string]string{
		"type":        taskType,
		"engineer_id": engineerID,
		"task_name":   name,
	}
	if engineerName := mustGetString(cmd, "engineer-name"); engineerName != "" {
		fields["engineer_name"] = engineerName
	}
	selection := mustGetString(cmd, "selection-json")
	if selection == "" {
		selection = `{"mode":"all"}`
	}
	fields["selection"] = selection
	files := map[string]string{}
	if taskType == "sca" {
		files["sca_vulnerabilities"] = reportFile
	} else {
		files["sast_vulnerabilities"] = reportFile
	}

	if sourceType == "zip" {
		zipFile := mustGetString(cmd, "zip-file")
		if zipFile == "" {
			return fmt.Errorf("please provide --zip-file")
		}
		if err := requireReadableFile(zipFile, "zip-file"); err != nil {
			return err
		}
		files["zip_file"] = zipFile
		if dryRun {
			return outputDryRun(cmd, []dryRunRequest{
				makeDryRunRequest(cfg, http.MethodPost, "/api/v1/codeforce/openapi/denoise-tasks/zip", nil, fields, files),
			})
		}
		data, err := client.openAPICreateDenoiseByZip(cmd.Context(), fields, files)
		if err != nil {
			return err
		}
		return outputDenoiseCreate(cmd, cfg, data, name, sourceType, taskType, engineerID)
	}

	branchOrTag := mustGetString(cmd, "branch-or-tag")
	if branchOrTag == "" {
		return fmt.Errorf("please provide --branch-or-tag")
	}
	repositoryName, err := resolveOpenAPIRepositoryName(cmd.Context(), client, mustGetString(cmd, "repository-id"), mustGetString(cmd, "repository-name"))
	if err != nil {
		return err
	}
	fields["repository_name"] = repositoryName
	fields["branch_or_tag"] = branchOrTag
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, "/api/v1/codeforce/openapi/denoise-tasks/repository", nil, fields, files),
		})
	}
	data, err := client.openAPICreateDenoiseByRepository(cmd.Context(), fields, files)
	if err != nil {
		return err
	}
	return outputDenoiseCreate(cmd, cfg, data, name, sourceType, taskType, engineerID)
}

func outputDenoiseCreate(cmd *cobra.Command, cfg Config, data map[string]any, name, sourceType, taskType, engineerID string) error {
	return outputOK(cmd, map[string]any{
		"task_id":      valueString(data["id"]),
		"task_name":    firstNonEmpty(valueString(data["name"]), valueString(data["task_name"]), name),
		"engineer_id":  firstNonEmpty(valueString(data["engineer_id"]), engineerID),
		"type":         firstNonEmpty(valueString(data["type"]), taskType),
		"source_type":  firstNonEmpty(valueString(data["source_type"]), sourceType),
		"status":       valueString(data["status"]),
		"account_type": cfg.AccountType,
		"denoise_task": data,
	})
}

func newDenoiseResultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "result",
		Short: "Get a denoise task result",
		RunE:  runDenoiseResult,
	}
	cmd.Flags().String("task-id", "", "Denoise task ID")
	cmd.AddCommand(newDenoiseResultDownloadCommand())
	return cmd
}

func runDenoiseResult(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	taskID := mustGetString(cmd, "task-id")
	if taskID == "" {
		return fmt.Errorf("please provide --task-id")
	}
	client := NewClient(cfg)
	if cfg.AccountType == accountTypeOpenAPI {
		data, err := client.openAPIDenoiseTaskStatus(cmd.Context(), taskID)
		if err != nil {
			return err
		}
		return outputOK(cmd, map[string]any{
			"task_id":      taskID,
			"account_type": cfg.AccountType,
			"denoise_task": data,
			"status":       valueString(data["status"]),
			"type":         valueString(data["type"]),
			"source_type":  valueString(data["source_type"]),
			"summary":      data["statistics"],
		})
	}
	task, err := client.denoiseResult(cmd.Context(), taskID)
	if err != nil {
		return err
	}
	stats, err := client.denoiseStatistics(cmd.Context(), taskID)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"task_id":      taskID,
		"account_type": cfg.AccountType,
		"status":       valueString(task["status"]),
		"type":         valueString(task["type"]),
		"source_type":  valueString(task["source_type"]),
		"denoise_task": task,
		"statistics":   stats,
	})
}

func newDenoiseResultDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download a denoise task export",
		RunE:  runDenoiseResultDownload,
	}
	cmd.Flags().String("task-id", "", "Denoise task ID")
	cmd.Flags().String("format", "json", "Export format (json|docx|pdf)")
	cmd.Flags().String("out", "", "Output file path")
	return cmd
}

func runDenoiseResultDownload(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if cfg.AccountType == accountTypeOpenAPI {
		return fmt.Errorf("denoise result download is not available with account_type=openapi because the public OpenAPI does not expose export endpoints")
	}
	taskID := mustGetString(cmd, "task-id")
	if taskID == "" {
		return fmt.Errorf("please provide --task-id")
	}
	format := strings.ToLower(mustGetString(cmd, "format"))
	switch format {
	case "json", "docx", "pdf":
	default:
		return fmt.Errorf("unsupported --format %q, use json, docx, or pdf", format)
	}
	client := NewClient(cfg)
	task, err := client.denoiseResult(cmd.Context(), taskID)
	if err != nil {
		return err
	}
	taskType := strings.ToLower(valueString(task["type"]))
	sca := taskType == "sca"
	ids, err := collectDenoiseVulnerabilityIDs(cmd.Context(), client, cfg, taskID, sca)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("no vulnerabilities found for task %s", taskID)
	}
	body := map[string]any{
		"vulnerability_ids": ids,
		"format":            format,
	}
	if dryRun {
		path := client.managementPath(
			"/api/v1/aiemployee/denoise/tasks/"+taskID+"/vulnerabilities/export",
			"/api/v1/user/aiemployee/denoise/tasks/"+taskID+"/vulnerabilities/export",
		)
		if sca {
			path = client.managementPath(
				"/api/v1/aiemployee/denoise/tasks/"+taskID+"/sca-vulnerabilities/export",
				"/api/v1/user/aiemployee/denoise/tasks/"+taskID+"/sca-vulnerabilities/export",
			)
		}
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, path, nil, body, nil),
		})
	}
	data, headers, err := client.denoiseExport(cmd.Context(), taskID, body, sca)
	if err != nil {
		return err
	}
	out := mustGetString(cmd, "out")
	if out == "" {
		base := sanitizeFilename(firstNonEmpty(valueString(task["name"]), taskID))
		out = base + "." + format
		if ext := strings.TrimPrefix(filepath.Ext(filenameFromHeader(headers, "")), "."); ext != "" {
			out = base + "." + ext
		}
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := writeOutputFile(out, data); err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"task_id":             taskID,
		"out":                 out,
		"format":              format,
		"vulnerability_count": len(ids),
		"bytes":               len(data),
	})
}

func newCodeManagementCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code-management",
		Short: "Manage CodeForce code packages",
	}
	cmd.AddCommand(newCodeManagementCreateCommand())
	return cmd
}

func newCodeManagementCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Upload a code package into CodeForce code management",
		RunE:  runCodeManagementCreate,
	}
	cmd.Flags().String("name", "", "Code repository name")
	cmd.Flags().String("description", "", "Code repository description")
	cmd.Flags().String("version-description", "", "Initial version description")
	cmd.Flags().String("file", "", "ZIP file path")
	return cmd
}

func runCodeManagementCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "code-management create"); err != nil {
		return err
	}
	name := mustGetString(cmd, "name")
	filePath := mustGetString(cmd, "file")
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if filePath == "" {
		return fmt.Errorf("please provide --file")
	}
	if err := requireReadableFile(filePath, "file"); err != nil {
		return err
	}
	fields := map[string]string{
		"name":                name,
		"description":         mustGetString(cmd, "description"),
		"version_description": mustGetString(cmd, "version-description"),
	}
	files := map[string]string{"file": filePath}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, managementPathForDryRun(cfg, "/api/v1/aiemployee/code-management", "/api/v1/user/code-management"), nil, fields, files),
		})
	}
	data, err := NewClient(cfg).codeManagementCreate(cmd.Context(), fields, files)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"code_repository_id":   valueString(data["id"]),
		"code_repository_name": firstNonEmpty(valueString(data["name"]), name),
		"description":          data["description"],
		"latest_version":       data["latest_version"],
		"version_count":        data["version_count"],
		"code_management":      data,
	})
}

func newRepositoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repository",
		Short: "Manage CodeForce repositories",
	}
	cmd.AddCommand(newRepositoryCreateCommand())
	return cmd
}

func newRepositoryCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create CodeForce repositories",
	}
	cmd.AddCommand(newRepositoryCreateProjectCommand())
	return cmd
}

func newRepositoryCreateProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create a project repository",
		RunE:  runRepositoryCreateProject,
	}
	cmd.Flags().String("name", "", "Repository name")
	cmd.Flags().String("platform", "", "Repository platform (github|gitlab|gitee|gitea)")
	cmd.Flags().String("repositories-url", "", "Git repository URL")
	cmd.Flags().String("repositories-user", "", "Repository username")
	cmd.Flags().String("token", "", "Repository access token")
	cmd.Flags().String("description", "", "Repository description")
	return cmd
}

func runRepositoryCreateProject(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "repository create project"); err != nil {
		return err
	}
	name := mustGetString(cmd, "name")
	platform, err := normalizePlatform(mustGetString(cmd, "platform"))
	if err != nil {
		return err
	}
	repositoryURL := mustGetString(cmd, "repositories-url")
	token := mustGetString(cmd, "token")
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if repositoryURL == "" {
		return fmt.Errorf("please provide --repositories-url")
	}
	if token == "" {
		return fmt.Errorf("please provide --token")
	}
	body := map[string]any{
		"name":              name,
		"platform":          platform,
		"repositories_url":  repositoryURL,
		"repositories_user": mustGetString(cmd, "repositories-user"),
		"token":             token,
		"description":       mustGetString(cmd, "description"),
		"repository_type":   "project",
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, managementPathForDryRun(cfg, "/api/v1/aiemployee/repository", "/api/v1/user/repository"), nil, body, nil),
		})
	}
	data, err := NewClient(cfg).repositoryCreate(cmd.Context(), body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"repository_id":    valueString(data["id"]),
		"repository_name":  firstNonEmpty(valueString(data["name"]), name),
		"platform":         firstNonEmpty(valueString(data["platform"]), platform),
		"repositories_url": firstNonEmpty(valueString(data["repositories_url"]), repositoryURL),
		"repository_type":  firstNonEmpty(valueString(data["repository_type"]), "project"),
		"repository":       data,
	})
}

func newGitAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-auth",
		Short: "Manage personal Git authorization configs",
	}
	cmd.AddCommand(newGitAuthCreateCommand())
	return cmd
}

func newGitAuthCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a personal Git authorization config",
		RunE:  runGitAuthCreate,
	}
	cmd.Flags().String("name", "", "Config name")
	cmd.Flags().String("platform", "", "Git platform (github|gitlab|gitee)")
	cmd.Flags().String("token", "", "Platform access token")
	cmd.Flags().String("base-url", "", "Base URL for self-hosted GitLab")
	return cmd
}

func runGitAuthCreate(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	if err := requireManagementAccount(cfg, "git-auth create"); err != nil {
		return err
	}
	name := mustGetString(cmd, "name")
	platform, err := normalizePlatform(mustGetString(cmd, "platform"))
	if err != nil {
		return err
	}
	token := mustGetString(cmd, "token")
	if name == "" {
		return fmt.Errorf("please provide --name")
	}
	if token == "" {
		return fmt.Errorf("please provide --token")
	}
	body := map[string]any{
		"name":     name,
		"platform": platform,
		"token":    token,
		"base_url": mustGetString(cmd, "base-url"),
	}
	if dryRun {
		return outputDryRun(cmd, []dryRunRequest{
			makeDryRunRequest(cfg, http.MethodPost, managementPathForDryRun(cfg, "/api/v1/codeforce/aiemployee/git-auth", "/api/v1/codeforce/user/git-auth"), nil, body, nil),
		})
	}
	data, err := NewClient(cfg).gitAuthCreate(cmd.Context(), body)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"config_id":       valueString(data["id"]),
		"config_name":     firstNonEmpty(valueString(data["name"]), name),
		"platform":        firstNonEmpty(valueString(data["platform"]), platform),
		"git_auth_config": data,
	})
}

func newOpenAPICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Inspect OpenAPI-accessible CodeForce resources",
	}
	cmd.AddCommand(newOpenAPIWhoAmICommand())
	cmd.AddCommand(newOpenAPIRepositoriesCommand())
	cmd.AddCommand(newOpenAPIDenoiseEngineersCommand())
	return cmd
}

func newOpenAPIWhoAmICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Verify the current OpenAPI key identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfigFromCommand(cmd)
			if err != nil {
				return err
			}
			data, err := NewClient(cfg).openAPIWhoAmI(cmd.Context())
			if err != nil {
				return err
			}
			return outputOK(cmd, map[string]any{
				"openapi_owner": data,
			})
		},
	}
}

func newOpenAPIRepositoriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repositories",
		Short: "List repositories visible to the current OpenAPI key",
		RunE:  runOpenAPIRepositories,
	}
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Int("size", defaultListPageSize, "Page size")
	cmd.Flags().String("platform", "", "Repository platform filter")
	cmd.Flags().String("keyword", "", "Repository keyword filter")
	return cmd
}

func runOpenAPIRepositories(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("page", strconv.Itoa(mustGetInt(cmd, "page")))
	query.Set("size", strconv.Itoa(mustGetInt(cmd, "size")))
	if platform := mustGetString(cmd, "platform"); platform != "" {
		query.Set("platform", platform)
	}
	if keyword := mustGetString(cmd, "keyword"); keyword != "" {
		query.Set("keyword", keyword)
	}
	data, err := NewClient(cfg).openAPIRepositories(cmd.Context(), query)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"repositories": data,
	})
}

func newOpenAPIDenoiseEngineersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoise-engineers",
		Short: "List denoise engineers visible to the current OpenAPI key",
		RunE:  runOpenAPIDenoiseEngineers,
	}
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Int("size", defaultListPageSize, "Page size")
	cmd.Flags().String("keyword", "", "Keyword filter")
	cmd.Flags().String("type", "", "Engineer type filter (sast|sca)")
	return cmd
}

func runOpenAPIDenoiseEngineers(cmd *cobra.Command, args []string) error {
	cfg, err := getConfigFromCommand(cmd)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("page", strconv.Itoa(mustGetInt(cmd, "page")))
	query.Set("size", strconv.Itoa(mustGetInt(cmd, "size")))
	if keyword := mustGetString(cmd, "keyword"); keyword != "" {
		query.Set("keyword", keyword)
	}
	if engineerType := mustGetString(cmd, "type"); engineerType != "" {
		query.Set("type", engineerType)
	}
	data, err := NewClient(cfg).openAPIDenoiseEngineers(cmd.Context(), query)
	if err != nil {
		return err
	}
	return outputOK(cmd, map[string]any{
		"denoise_engineers": data,
	})
}

func requireManagementAccount(cfg Config, commandName string) error {
	if cfg.AccountType == accountTypeOpenAPI {
		return fmt.Errorf("%s requires account_type=admin or account_type=user; the current token type should use account_type=openapi only for public OpenAPI endpoints", commandName)
	}
	return nil
}

func normalizeDenoiseType(taskType string) (string, error) {
	taskType = strings.ToLower(strings.TrimSpace(taskType))
	switch taskType {
	case "", "sast":
		return "sast", nil
	case "sca":
		return "sca", nil
	default:
		return "", fmt.Errorf("unsupported denoise type %q, use sast or sca", taskType)
	}
}

func normalizePlatform(platform string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	switch platform {
	case "github", "gitlab", "gitee", "gitea":
		return platform, nil
	default:
		return "", fmt.Errorf("unsupported platform %q, use github, gitlab, gitee, or gitea", platform)
	}
}

func requireReadableFile(path, field string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s %s: %w", field, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s %s is a directory, expected a file", field, path)
	}
	return nil
}

func managementPathForDryRun(cfg Config, adminPath, userPath string) string {
	if cfg.AccountType == accountTypeUser {
		return userPath
	}
	return adminPath
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func filenameFromHeader(headers http.Header, fallback string) string {
	if headers == nil {
		return fallback
	}
	contentDisposition := headers.Get("Content-Disposition")
	if contentDisposition == "" {
		return fallback
	}
	for _, part := range strings.Split(contentDisposition, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "filename=") {
			name := strings.Trim(strings.TrimSpace(part[len("filename="):]), `"`)
			if name != "" {
				return name
			}
		}
	}
	return fallback
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "codeforce-export"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	return replacer.Replace(name)
}

func resolveOpenAPIRepositoryName(ctx context.Context, client *Client, repositoryID, repositoryName string) (string, error) {
	if strings.TrimSpace(repositoryName) != "" {
		return strings.TrimSpace(repositoryName), nil
	}
	if strings.TrimSpace(repositoryID) == "" {
		return "", fmt.Errorf("please provide --repository-name or --repository-id")
	}
	page := 1
	for {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("size", strconv.Itoa(defaultListPageSize))
		data, err := client.openAPIRepositories(ctx, query)
		if err != nil {
			return "", err
		}
		items := mapSlice(data["items"])
		for _, item := range items {
			if valueString(item["id"]) == repositoryID {
				return firstNonEmpty(valueString(item["name"]), valueString(item["repository_name"])), nil
			}
		}
		if len(items) < defaultListPageSize {
			break
		}
		page++
	}
	return "", fmt.Errorf("repository id %s is not visible through the current OpenAPI key; provide --repository-name explicitly", repositoryID)
}

func collectDenoiseVulnerabilityIDs(ctx context.Context, client *Client, cfg Config, taskID string, sca bool) ([]string, error) {
	ids := []string{}
	page := 1
	for {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("size", strconv.Itoa(defaultListPageSize))
		var data map[string]any
		var err error
		if sca {
			path := client.managementPath(
				fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/sca-vulnerabilities", taskID),
				fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/sca-vulnerabilities", taskID),
			)
			err = client.DoJSON(ctx, http.MethodGet, path, query, nil, &data)
		} else {
			path := client.managementPath(
				fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/vulnerabilities", taskID),
				fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/vulnerabilities", taskID),
			)
			err = client.DoJSON(ctx, http.MethodGet, path, query, nil, &data)
		}
		if err != nil {
			return nil, err
		}
		items := mapSlice(data["items"])
		for _, item := range items {
			if id := valueString(item["id"]); id != "" {
				ids = append(ids, id)
			}
		}
		if !hasNextPage(data) || len(items) == 0 {
			break
		}
		page++
	}
	return ids, nil
}

func hasNextPage(data map[string]any) bool {
	pageInfo, _ := data["page_info"].(map[string]any)
	if pageInfo != nil {
		if next, ok := pageInfo["has_next_page"].(bool); ok {
			return next
		}
	}
	return false
}

func mapSlice(value any) []map[string]any {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if item, ok := row.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}
