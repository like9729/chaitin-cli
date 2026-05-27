package apisec

import "testing"

func TestParseOpenAPI(t *testing.T) {
	api, err := parseOpenAPI([]byte(`{"openapi":"3.0.3","info":{"title":"APISec","version":"26.05"},"paths":{"/api/ApplicationAPI":{"get":{"operationId":"ApplicationAPI_get","summary":"List applications"}}}}`))
	if err != nil {
		t.Fatalf("parseOpenAPI() error = %v", err)
	}
	if api.Paths["/api/ApplicationAPI"].Get.OperationID != "ApplicationAPI_get" {
		t.Fatalf("operationId not parsed: %+v", api.Paths["/api/ApplicationAPI"].Get)
	}
}

func TestParseCLIMapping(t *testing.T) {
	mapping, err := parseCLIMapping([]byte(`commands:
  - path: [asset, app, list]
    operationId: ApplicationAPI_get
    short: List applications
`))
	if err != nil {
		t.Fatalf("parseCLIMapping() error = %v", err)
	}
	if got := mapping.Commands[0].Path[2]; got != "list" {
		t.Fatalf("path[2] = %q, want list", got)
	}
}

func TestLoadEmbeddedSchema(t *testing.T) {
	api, mapping, err := loadEmbeddedSchema()
	if err != nil {
		t.Fatalf("loadEmbeddedSchema() error = %v", err)
	}
	if len(api.Paths) == 0 {
		t.Fatalf("embedded OpenAPI has no paths")
	}
	if mapping == nil {
		t.Fatalf("mapping is nil")
	}
}
