package site

import (
	"fmt"
	"io"

	"github.com/chaitin/chaitin-cli/products/safeline/pkg/client"
)

type createCheckResult struct {
	OK        bool            `json:"ok"`
	Operation string          `json:"operation"`
	Warnings  []string        `json:"warnings"`
	Errors    []string        `json:"errors"`
	Data      createCheckData `json:"data"`
}

type createCheckData struct {
	Endpoint     string         `json:"endpoint"`
	Payload      map[string]any `json:"payload"`
	RemoteChecks []string       `json:"remote_checks"`
}

func newCheckResult(endpoint string, payload map[string]any, warnings, errors []string) createCheckResult {
	return createCheckResult{OK: len(errors) == 0, Operation: "site.create.check", Warnings: warnings, Errors: errors, Data: createCheckData{Endpoint: endpoint, Payload: payload}}
}

func localCreateChecks(payload map[string]any) ([]string, []string) {
	warnings := []string{}
	errors := []string{}
	seen := map[string]bool{}
	ports, ok := normalizePortObjects(payload["ports"])
	if !ok {
		return warnings, append(errors, "ports must be a list of objects")
	}
	for _, p := range ports {
		port := fmt.Sprint(p["port"])
		ssl := fmt.Sprint(p["ssl"])
		key := port + ":" + ssl
		if seen[key] {
			warnings = append(warnings, "duplicate port entry in request; backend will reject invalid duplicates")
		}
		seen[key] = true
		opposite := port + ":" + fmt.Sprint(!truthy(p["ssl"]))
		if seen[opposite] {
			warnings = append(warnings, "same request mixes SSL and non-SSL on one port; backend usually rejects this")
		}
	}
	return warnings, errors
}

func normalizePortObjects(v any) ([]map[string]any, bool) {
	switch ports := v.(type) {
	case []map[string]any:
		return ports, true
	case []any:
		out := make([]map[string]any, 0, len(ports))
		for _, p := range ports {
			m, ok := p.(map[string]any)
			if !ok {
				return nil, false
			}
			out = append(out, m)
		}
		return out, true
	default:
		return nil, false
	}
}

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

type readOnlyClient interface {
	Do(method, path string, body io.Reader, query map[string]string) (*client.Envelope, error)
}

func remoteCreateChecks(c readOnlyClient, endpoint string, payload map[string]any) []string {
	checks := []string{}
	if certID := idString(payload["ssl_cert"]); certID != "" && certID != "0" {
		if _, err := c.Do("GET", "/api/CertAPI", nil, map[string]string{"id": certID}); err != nil {
			checks = append(checks, "certificate lookup failed; backend create remains final authority: "+err.Error())
		} else {
			checks = append(checks, "certificate lookup passed")
		}
	}
	if policyGroupID := idString(payload["policy_group"]); policyGroupID != "" && policyGroupID != "0" {
		if _, err := c.Do("GET", "/api/PolicyGroupAPI", nil, map[string]string{"id": policyGroupID}); err != nil {
			checks = append(checks, "policy group lookup failed; backend create remains final authority: "+err.Error())
		} else {
			checks = append(checks, "policy group lookup passed")
		}
	}
	if _, err := c.Do("GET", endpoint, nil, nil); err != nil {
		checks = append(checks, "site list lookup failed; duplicate check skipped: "+err.Error())
	} else {
		checks = append(checks, "site list lookup passed; backend still performs authoritative duplicate checks")
	}
	return checks
}

func idString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case map[string]any:
		return fmt.Sprint(t["id"])
	default:
		return fmt.Sprint(t)
	}
}
