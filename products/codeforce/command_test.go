package codeforce

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaitin/chaitin-cli/config"
	"gopkg.in/yaml.v3"
)

func TestNewCommandHelpShowsGroups(t *testing.T) {
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	help := out.String()
	for _, want := range []string{"project", "audit", "denoise", "repository", "git-auth", "--account-type"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestOpenAPIHelpShowsDiscoveryCommands(t *testing.T) {
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"openapi", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	help := out.String()
	for _, want := range []string{"whoami", "repositories", "denoise-engineers"} {
		if !strings.Contains(help, want) {
			t.Fatalf("openapi help missing %q:\n%s", want, help)
		}
	}
}

func TestDryRunUsesEnvConfigAndRedactsSecrets(t *testing.T) {
	t.Setenv("CODEFORCE_URL", "https://cf.example.com")
	t.Setenv("CODEFORCE_ACCESS_TOKEN", "cf-secret")
	t.Setenv("CODEFORCE_ACCOUNT_TYPE", "admin")
	withRuntime(t, config.Raw{}, true)

	cmd := NewCommand()
	cmd.SetArgs([]string{
		"repository", "create", "project",
		"--name", "demo-git",
		"--platform", "gitlab",
		"--repositories-url", "https://git.example.com/group/demo-git.git",
		"--token", "repo-secret",
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(out.String(), "cf-secret") || strings.Contains(out.String(), "repo-secret") {
		t.Fatalf("dry-run output leaked secret:\n%s", out.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("dry-run output is not JSON: %v\n%s", err, out.String())
	}
	requests := payload["requests"].([]any)
	req := requests[0].(map[string]any)
	if req["path"] != "/api/v1/aiemployee/repository" {
		t.Fatalf("path = %v", req["path"])
	}
}

func TestProjectCreateSendsManagementBody(t *testing.T) {
	var createdBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertAuth(t, r)
		if r.URL.Path != "/api/v1/codeforce/projects" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		decodeJSON(t, r.Body, &createdBody)
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"id":                "project-1",
			"name":              "demo-app",
			"repository_id":     "repo-1",
			"current_user_role": "owner",
		}})
	}))
	defer server.Close()
	withRuntime(t, rawConfig(Config{URL: server.URL, AccessToken: "cf-token", AccountType: accountTypeAdmin}), false)

	cmd := NewCommand()
	cmd.SetArgs([]string{"project", "create", "--name", "demo-app", "--repository-id", "repo-1", "--description", "demo"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if createdBody["name"] != "demo-app" || createdBody["repository_id"] != "repo-1" {
		t.Fatalf("unexpected project body: %#v", createdBody)
	}
	if !strings.Contains(out.String(), `"project_id": "project-1"`) {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestDenoiseParseUsesOpenAPIEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")
	writeFile(t, reportPath, `{"demo":true}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertAuth(t, r)
		if r.URL.Path != "/api/v1/codeforce/openapi/denoise-tasks/parse" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.MultipartForm.Value["type"][0]; got != "sast" {
			t.Fatalf("type = %q", got)
		}
		assertMultipartFile(t, r.MultipartForm, "file", "report.json")
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"type":                                  "sast",
			"max_selected_vulnerabilities_per_task": 50,
			"vulnerabilities": []any{
				map[string]any{"vulnerability_id": "v1", "name": "SQL Injection"},
			},
		}})
	}))
	defer server.Close()
	withRuntime(t, rawConfig(Config{URL: server.URL, AccessToken: "cf-token", AccountType: accountTypeOpenAPI}), false)

	cmd := NewCommand()
	cmd.SetArgs([]string{"denoise", "parse", "--type", "sast", "--report-file", reportPath})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), `"type": "sast"`) || !strings.Contains(out.String(), `"vulnerability_id": "v1"`) {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestCodeManagementCreateUsesMultipart(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "demo.zip")
	writeFile(t, zipPath, "zip-bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertAuth(t, r)
		if r.URL.Path != "/api/v1/aiemployee/code-management" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.MultipartForm.Value["name"][0]; got != "demo-code-drop" {
			t.Fatalf("name = %q", got)
		}
		assertMultipartFile(t, r.MultipartForm, "file", "demo.zip")
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"id":             "code-1",
			"name":           "demo-code-drop",
			"latest_version": 1,
			"version_count":  1,
		}})
	}))
	defer server.Close()
	withRuntime(t, rawConfig(Config{URL: server.URL, AccessToken: "cf-token", AccountType: accountTypeAdmin}), false)

	cmd := NewCommand()
	cmd.SetArgs([]string{
		"code-management", "create",
		"--name", "demo-code-drop",
		"--description", "release package",
		"--version-description", "2026-06 release",
		"--file", zipPath,
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), `"code_repository_id": "code-1"`) {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func withRuntime(t *testing.T, raw config.Raw, isDryRun bool) {
	t.Helper()
	runtimeCfg = Config{}
	dryRun = false
	ApplyRuntimeConfig(nil, raw, isDryRun)
	t.Cleanup(func() {
		runtimeCfg = Config{}
		dryRun = false
	})
}

func rawConfig(cfg Config) config.Raw {
	var node yaml.Node
	_ = node.Encode(cfg)
	return config.Raw{productName: node}
}

func assertAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer cf-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := r.Header.Get("X-API-Key"); got != "cf-token" {
		t.Fatalf("X-API-Key = %q", got)
	}
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertMultipartFile(t *testing.T, form *multipart.Form, field, wantFilename string) {
	t.Helper()
	files := form.File[field]
	if len(files) != 1 {
		t.Fatalf("field %s file count = %d, want 1", field, len(files))
	}
	if files[0].Filename != wantFilename {
		t.Fatalf("field %s filename = %q, want %q", field, files[0].Filename, wantFilename)
	}
}
