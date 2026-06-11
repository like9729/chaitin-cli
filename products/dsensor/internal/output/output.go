package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/chaitin/chaitin-cli/products/dsensor/internal/client"
)

// Format renders an API response in the specified format.
func Format(resp *client.APIResponse, format string) (string, error) {
	switch format {
	case "json":
		return formatJSON(resp)
	case "table":
		return formatTable(resp)
	default:
		return "", fmt.Errorf("不支持的输出格式: %s (支持: json, table)", format)
	}
}

// Output writes the formatted response to stdout.
func Output(resp *client.APIResponse, format string) error {
	out, err := Format(resp, format)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func formatJSON(resp *client.APIResponse) (string, error) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON 序列化失败: %w", err)
	}
	return string(data), nil
}

func formatTable(resp *client.APIResponse) (string, error) {
	if resp.Err != "" {
		return "", fmt.Errorf("API 错误: %s (%s)", resp.Err, resp.Msg)
	}

	if resp.Data == nil {
		return "(空响应)", nil
	}

	// Try to unmarshal data
	var rawData interface{}
	if err := json.Unmarshal(resp.Data, &rawData); err != nil {
		return string(resp.Data), nil
	}

	switch data := rawData.(type) {
	case []interface{}:
		return formatArrayTable(data)
	case map[string]interface{}:
		return formatObjectTable(data)
	default:
		return fmt.Sprintf("%v", rawData), nil
	}
}

func formatArrayTable(arr []interface{}) (string, error) {
	if len(arr) == 0 {
		return "(空列表)", nil
	}

	if obj, ok := arr[0].(map[string]interface{}); ok {
		keys := sortedKeys(obj)
		return renderTable(arr, keys)
	}

	// Simple value array
	var lines []string
	for _, item := range arr {
		lines = append(lines, fmt.Sprintf("  - %v", item))
	}
	return strings.Join(lines, "\n"), nil
}

func formatObjectTable(obj map[string]interface{}) (string, error) {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	for _, k := range sortedKeys(obj) {
		v := obj[k]
		// Skip nested objects/arrays for direct table
		switch val := v.(type) {
		case map[string]interface{}, []interface{}:
			jsonVal, _ := json.Marshal(val)
			fmt.Fprintf(w, "%s:\t%s\n", k, string(jsonVal))
		default:
			fmt.Fprintf(w, "%s:\t%v\n", k, val)
		}
	}
	w.Flush()
	return buf.String(), nil
}

func renderTable(arr []interface{}, keys []string) (string, error) {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintf(w, "%s\n", strings.Join(keys, "\t"))

	// Separator
	var seps []string
	for range keys {
		seps = append(seps, "---")
	}
	fmt.Fprintf(w, "%s\n", strings.Join(seps, "\t"))

	// Rows
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var row []string
		for _, k := range keys {
			val := obj[k]
			switch v := val.(type) {
			case map[string]interface{}, []interface{}:
				jsonVal, _ := json.Marshal(v)
				row = append(row, string(jsonVal))
			case nil:
				row = append(row, "")
			default:
				row = append(row, fmt.Sprintf("%v", v))
			}
		}
		fmt.Fprintf(w, "%s\n", strings.Join(row, "\t"))
	}

	w.Flush()
	return buf.String(), nil
}

func sortedKeys(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	// Simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
