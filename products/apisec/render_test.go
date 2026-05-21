package apisec

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRendererWritesJSON(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(FormatJSON, &out)

	if err := renderer.Render(map[string]any{"ok": true}); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("ok = %#v, want true", got["ok"])
	}
}

func TestRendererWritesTableAsStableJSON(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(FormatTable, &out)

	if err := renderer.Render(map[string]any{"ok": true}); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got := out.String(); got != "{\n  \"ok\": true\n}\n" {
		t.Fatalf("output = %q, want indented JSON", got)
	}
}
