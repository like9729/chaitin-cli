package safeline3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type requestOptions struct {
	Query       []string
	Request     string
	RequestFile string
}

type writeOptions struct {
	Check   bool
	Explain bool
	Yes     bool
}

func addWriteFlags(cmd *cobra.Command, opts *writeOptions) {
	cmd.Flags().BoolVar(&opts.Check, "check", false, "Validate and print the generated request without writing")
	cmd.Flags().BoolVar(&opts.Explain, "explain", false, "Explain the generated request without writing")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Confirm this write operation")
}

func addRequestFlags(cmd *cobra.Command, opts *requestOptions, query bool, body bool) {
	if query {
		cmd.Flags().StringArrayVar(&opts.Query, "param", nil, "Query parameter in key=value form; can be repeated")
	}
	if body {
		cmd.Flags().StringVar(&opts.Request, "body", "", "JSON request body")
		cmd.Flags().StringVar(&opts.RequestFile, "body-file", "", "Path to JSON request body file, or - for stdin")
	}
}

func doRequest(cmd *cobra.Command, method, path string, query url.Values, body any) error {
	var result any
	if err := getClient(cmd).Do(context.Background(), method, path, query, body, &result); err != nil {
		if _, ok := err.(dryRunResult); ok {
			return nil
		}
		return err
	}
	return getRenderer(cmd).Render(result)
}

func doWrite(cmd *cobra.Command, opts writeOptions, operation, method, path string, query url.Values, body any, notes []string) error {
	if opts.Check || opts.Explain {
		return getRenderer(cmd).Render(map[string]any{
			"ok":        true,
			"operation": operation,
			"dry_run":   true,
			"method":    method,
			"path":      path,
			"query":     query,
			"body":      body,
			"notes":     notes,
		})
	}
	if !opts.Yes {
		return fmt.Errorf("%s is a write operation; re-run with --yes or use --check to preview", operation)
	}
	return doRequest(cmd, method, path, query, body)
}

func parseQuery(items []string) (url.Values, error) {
	values := url.Values{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid query %q, expected key=value", item)
		}
		values.Add(strings.TrimSpace(key), value)
	}
	return values, nil
}

func readRequestBody(opts requestOptions) (any, error) {
	if opts.Request != "" && opts.RequestFile != "" {
		return nil, fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	if opts.Request != "" {
		return parseJSON([]byte(opts.Request))
	}
	if opts.RequestFile == "" {
		return nil, nil
	}
	return readJSONFile(opts.RequestFile)
}

func readJSONFile(path string) (any, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read JSON file: %w", err)
	}
	return parseJSON(data)
}

func parseJSON(data []byte) (any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return value, nil
}

func splitValues(values []string) []string {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func parseUintList(values []string) ([]uint64, error) {
	parts := splitValues(values)
	out := make([]uint64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("invalid ID %q", part)
		}
		out = append(out, id)
	}
	return out, nil
}

func parseIDs(args []string) ([]uint64, error) {
	return parseUintList(args)
}

func parseID(arg string) (uint64, error) {
	id, err := strconv.ParseUint(arg, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid ID %q", arg)
	}
	return id, nil
}

func addPagination(query url.Values, page, pageSize int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	query.Set("page", strconv.Itoa(page))
	query.Set("page_size", strconv.Itoa(pageSize))
	query.Set("offset", strconv.Itoa((page-1)*pageSize))
	query.Set("count", strconv.Itoa(pageSize))
}

func parseTimeValue(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "now" {
		return time.Now().UnixMilli(), nil
	}
	if strings.HasPrefix(raw, "-") {
		d, err := time.ParseDuration(strings.TrimPrefix(raw, "-"))
		if err != nil {
			return 0, fmt.Errorf("invalid relative time %q", raw)
		}
		return time.Now().Add(-d).UnixMilli(), nil
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if n < 1_000_000_000_000 {
			return n * 1000, nil
		}
		return n, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("invalid time %q", raw)
}

func normalizeType(t string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "reverse-proxy", "reverseproxy", "":
		return "reverse-proxy", nil
	case "route-proxy", "routeproxy":
		return "route-proxy", nil
	case "transparent-proxy", "transparentproxy":
		return "transparent-proxy", nil
	case "transparent":
		return "transparent", nil
	case "mirror":
		return "mirror", nil
	case "sdk":
		return "sdk", nil
	default:
		return "", fmt.Errorf("invalid type %q", t)
	}
}

func apiProtectType(t string) (string, error) {
	t, err := normalizeType(t)
	if err != nil {
		return "", err
	}
	switch t {
	case "reverse-proxy":
		return "ReverseProxy", nil
	case "route-proxy":
		return "RouteProxy", nil
	case "transparent-proxy":
		return "TransparentProxy", nil
	case "transparent":
		return "Transparent", nil
	case "mirror":
		return "Mirror", nil
	case "sdk":
		return "SDK", nil
	default:
		return "", fmt.Errorf("invalid type %q", t)
	}
}

func pathForType(t string) (string, error) {
	t, err := normalizeType(t)
	if err != nil {
		return "", err
	}
	return "/api/v3/protected-object/" + t, nil
}

func normalizeMode(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "reverse-proxy", "reverseproxy", "reverse":
		return "ReverseProxy", nil
	case "route-proxy", "routeproxy", "router", "route":
		return "RouteProxy", nil
	case "transparent":
		return "Transparent", nil
	case "mirror":
		return "Mirror", nil
	case "sdk":
		return "SDK", nil
	default:
		return "", fmt.Errorf("invalid mode %q", mode)
	}
}

func modeCLI(mode string) string {
	switch mode {
	case "ReverseProxy":
		return "reverse-proxy"
	case "RouteProxy":
		return "route-proxy"
	case "Transparent":
		return "transparent"
	case "Mirror":
		return "mirror"
	case "SDK":
		return "sdk"
	default:
		return strings.ToLower(mode)
	}
}

func supportedSiteTypes(mode string) []string {
	switch mode {
	case "ReverseProxy":
		return []string{"reverse-proxy"}
	case "RouteProxy":
		return []string{"route-proxy"}
	case "Transparent":
		return []string{"transparent-proxy", "transparent"}
	case "Mirror":
		return []string{"transparent", "mirror"}
	case "SDK":
		return []string{"sdk"}
	default:
		return nil
	}
}

func isTypeSupportedByMode(t, mode string) bool {
	t, _ = normalizeType(t)
	for _, supported := range supportedSiteTypes(mode) {
		if supported == t {
			return true
		}
	}
	return false
}

func stateValue(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "detect":
		return "Detect", nil
	case "dry-run", "dryrun":
		return "DryRun", nil
	case "bypass", "by-pass":
		return "Bypass", nil
	case "forbidden":
		return "Forbidden", nil
	case "not-apply", "notapply":
		return "NotApply", nil
	case "redirect":
		return "Redirect", nil
	case "response":
		return "Response", nil
	case "cache":
		return "Cache", nil
	default:
		return "", fmt.Errorf("invalid state %q", raw)
	}
}

func urlPathOp(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "prefix", "pre":
		return "pre", nil
	case "exact":
		return "exact", nil
	case "regex", "reg":
		return "reg", nil
	default:
		return "", fmt.Errorf("invalid url path op %q", raw)
	}
}

func boolPtr(v bool) *bool { return &v }

func queryFromMap(values map[string]string) url.Values {
	q := url.Values{}
	for k, v := range values {
		if v != "" {
			q.Set(k, v)
		}
	}
	return q
}

func equalCondition(values ...any) map[string]any {
	return map[string]any{"operator": "=", "value": values}
}

func boolCondition(value bool) map[string]any {
	return map[string]any{"operator": "=", "value": value}
}

func valueCondition(value any) map[string]any {
	return map[string]any{"value": value}
}

func httpMethodForEnabled(enabled bool) string {
	_ = enabled
	return http.MethodPut
}
