package codeforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) managementPath(adminPath, userPath string) string {
	if c.config.AccountType == accountTypeUser {
		return userPath
	}
	return adminPath
}

func (c *Client) projectCreate(ctx context.Context, body map[string]any) (map[string]any, error) {
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodPost, "/api/v1/codeforce/projects", nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) projectAIEmployeeModelOptions(ctx context.Context, projectID, employeeID string) (map[string]any, error) {
	query := url.Values{}
	if strings.TrimSpace(employeeID) != "" {
		query.Set("employee_id", employeeID)
	}
	var data map[string]any
	path := fmt.Sprintf("/api/v1/codeforce/projects/%s/ai-employees/model-options", projectID)
	if err := c.DoJSON(ctx, http.MethodGet, path, query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) projectAIEmployeeCreate(ctx context.Context, projectID string, body map[string]any) (map[string]any, error) {
	var data map[string]any
	path := fmt.Sprintf("/api/v1/codeforce/projects/%s/ai-employees", projectID)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) projectAIDevCreate(ctx context.Context, projectID string, body map[string]any) (map[string]any, error) {
	var data map[string]any
	path := fmt.Sprintf("/api/v1/codeforce/projects/%s/ai-dev/tasks", projectID)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) projectAIDevResult(ctx context.Context, projectID, taskID string) (map[string]any, error) {
	var data map[string]any
	path := fmt.Sprintf("/api/v1/codeforce/projects/%s/ai-dev/tasks/%s", projectID, taskID)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) nativeAuditCreate(ctx context.Context, body map[string]any) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath("/api/v1/aiemployee/aitask/manual", "/api/v1/user/aiemployee/aitask/manual")
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) nativeAuditResult(ctx context.Context, taskID string) (map[string]any, error) {
	query := url.Values{}
	query.Set("id", taskID)
	var data map[string]any
	path := c.managementPath("/api/v1/aiemployee/aitask/info", "/api/v1/user/aiemployee/aitask/info")
	if err := c.DoJSON(ctx, http.MethodGet, path, query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) manualOverviewExport(ctx context.Context, query url.Values) ([]byte, http.Header, error) {
	path := c.managementPath("/api/v1/aiemployee/aitask/manual-overview/export", "/api/v1/user/aiemployee/aitask/manual-overview/export")
	return c.Download(ctx, http.MethodGet, path, query, nil)
}

func (c *Client) denoiseCreate(ctx context.Context, fields map[string]string, files map[string]string, isZip bool) (map[string]any, error) {
	var data map[string]any
	if isZip {
		path := c.managementPath("/api/v1/aiemployee/denoise/tasks", "/api/v1/user/aiemployee/denoise/tasks")
		if err := c.DoMultipart(ctx, http.MethodPost, path, fields, files, &data); err != nil {
			return nil, err
		}
		return data, nil
	}
	body := make(map[string]any, len(fields))
	for key, value := range fields {
		switch key {
		case "vulnerabilities", "sca_vulnerabilities":
			if strings.TrimSpace(value) == "" {
				continue
			}
			var parsed any
			if err := jsonUnmarshalString(value, &parsed); err != nil {
				return nil, fmt.Errorf("parse %s: %w", key, err)
			}
			body[key] = parsed
		default:
			body[key] = value
		}
	}
	path := c.managementPath("/api/v1/aiemployee/denoise/tasks", "/api/v1/user/aiemployee/denoise/tasks")
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) denoiseParse(ctx context.Context, taskType, reportFile string) (*reportParseResult, error) {
	var data reportParseResult
	path := c.managementPath("/api/v1/aiemployee/denoise/tasks/parse", "/api/v1/user/aiemployee/denoise/tasks/parse")
	fields := map[string]string{"type": taskType}
	files := map[string]string{"file": reportFile}
	if err := c.DoMultipart(ctx, http.MethodPost, path, fields, files, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) denoiseResult(ctx context.Context, taskID string) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath(fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s", taskID), fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s", taskID))
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) denoiseStatistics(ctx context.Context, taskID string) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath(
		fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/statistics", taskID),
		fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/statistics", taskID),
	)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) denoiseVulnerabilities(ctx context.Context, taskID string) (map[string]any, error) {
	var data map[string]any
	query := url.Values{}
	query.Set("page", "1")
	query.Set("size", strconv.Itoa(defaultListPageSize))
	path := c.managementPath(
		fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/vulnerabilities", taskID),
		fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/vulnerabilities", taskID),
	)
	if err := c.DoJSON(ctx, http.MethodGet, path, query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) denoiseScaVulnerabilities(ctx context.Context, taskID string) (map[string]any, error) {
	var data map[string]any
	query := url.Values{}
	query.Set("page", "1")
	query.Set("size", strconv.Itoa(defaultListPageSize))
	path := c.managementPath(
		fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/sca-vulnerabilities", taskID),
		fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/sca-vulnerabilities", taskID),
	)
	if err := c.DoJSON(ctx, http.MethodGet, path, query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) denoiseExport(ctx context.Context, taskID string, body map[string]any, sca bool) ([]byte, http.Header, error) {
	var path string
	if sca {
		path = c.managementPath(
			fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/sca-vulnerabilities/export", taskID),
			fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/sca-vulnerabilities/export", taskID),
		)
	} else {
		path = c.managementPath(
			fmt.Sprintf("/api/v1/aiemployee/denoise/tasks/%s/vulnerabilities/export", taskID),
			fmt.Sprintf("/api/v1/user/aiemployee/denoise/tasks/%s/vulnerabilities/export", taskID),
		)
	}
	return c.Download(ctx, http.MethodPost, path, nil, body)
}

func (c *Client) codeManagementCreate(ctx context.Context, fields map[string]string, files map[string]string) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath("/api/v1/aiemployee/code-management", "/api/v1/user/code-management")
	if err := c.DoMultipart(ctx, http.MethodPost, path, fields, files, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) repositoryCreate(ctx context.Context, body map[string]any) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath("/api/v1/aiemployee/repository", "/api/v1/user/repository")
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) gitAuthCreate(ctx context.Context, body map[string]any) (map[string]any, error) {
	var data map[string]any
	path := c.managementPath("/api/v1/codeforce/aiemployee/git-auth", "/api/v1/codeforce/user/git-auth")
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIWhoAmI(ctx context.Context) (map[string]any, error) {
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/whoami", nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIRepositories(ctx context.Context, query url.Values) (map[string]any, error) {
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/repositories", query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIDenoiseEngineers(ctx context.Context, query url.Values) (map[string]any, error) {
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/denoise-engineers", query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIParseDenoiseReport(ctx context.Context, fields map[string]string, files map[string]string) (*reportParseResult, error) {
	var data reportParseResult
	if err := c.DoMultipart(ctx, http.MethodPost, "/api/v1/codeforce/openapi/denoise-tasks/parse", fields, files, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) openAPICreateDenoiseByRepository(ctx context.Context, fields map[string]string, files map[string]string) (map[string]any, error) {
	var data map[string]any
	if err := c.DoMultipart(ctx, http.MethodPost, "/api/v1/codeforce/openapi/denoise-tasks/repository", fields, files, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPICreateDenoiseByZip(ctx context.Context, fields map[string]string, files map[string]string) (map[string]any, error) {
	var data map[string]any
	if err := c.DoMultipart(ctx, http.MethodPost, "/api/v1/codeforce/openapi/denoise-tasks/zip", fields, files, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIDenoiseTaskStatus(ctx context.Context, taskID string) (map[string]any, error) {
	query := url.Values{}
	query.Set("task_id", taskID)
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/denoise-tasks/status", query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIAIAuditTaskStatus(ctx context.Context, taskID string) (map[string]any, error) {
	query := url.Values{}
	query.Set("task_id", taskID)
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/ai-audit-tasks/status", query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) openAPIManualAIAuditTaskStatus(ctx context.Context, taskID string) (map[string]any, error) {
	query := url.Values{}
	query.Set("task_id", taskID)
	var data map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/codeforce/openapi/manual-ai-audit-tasks/status", query, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func jsonUnmarshalString(raw string, target any) error {
	return json.Unmarshal([]byte(raw), target)
}
