package safeline3

import (
	"fmt"
	"strings"
)

func filterMaps(items []map[string]any, keep func(map[string]any) bool) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func stringField(item map[string]any, key string) string {
	if item == nil {
		return ""
	}
	switch v := item[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func boolField(item map[string]any, key string) bool {
	if item == nil {
		return false
	}
	switch v := item[key].(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}

func containsFold(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}
