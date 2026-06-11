package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseSpec unmarshals raw JSON bytes into a Spec.
func ParseSpec(data []byte) (*Spec, error) {
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}
	return &spec, nil
}

// FlattenCommands extracts all operations from the spec into CommandSpecs.
// It resolves $ref references inline before extracting parameters.
func FlattenCommands(spec *Spec) ([]CommandSpec, error) {
	schemaIndex := buildSchemaIndex(spec)

	var commands []CommandSpec
	for path, pathItem := range spec.Paths {
		for _, opEntry := range []struct {
			method string
			op     *Operation
		}{
			{"POST", pathItem.Post},
			{"GET", pathItem.Get},
			{"PUT", pathItem.Put},
		} {
			if opEntry.op == nil {
				continue
			}
			cmd := CommandSpec{
				OperationID: opEntry.op.OperationID,
				Summary:     opEntry.op.Summary,
				Tags:        opEntry.op.Tags,
				Path:        path,
				Method:      opEntry.method,
			}

			if opEntry.op.RequestBody != nil {
				if schema, ok := getJSONSchema(opEntry.op.RequestBody); ok {
					resolved := resolveSchema(schema, schemaIndex)
					cmd.BodyParams = extractParams(resolved)
					cmd.HasBody = len(cmd.BodyParams) > 0 || hasProperties(resolved)
				}
			}

			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

// buildSchemaIndex creates a map from $ref name to resolved Schema.
func buildSchemaIndex(spec *Spec) map[string]*Schema {
	index := make(map[string]*Schema)
	if spec.Components.Schemas == nil {
		return index
	}
	for name, schema := range spec.Components.Schemas {
		s := schema
		index["#/components/schemas/"+name] = &s
	}
	return index
}

// getJSONSchema extracts the JSON request body schema if present.
func getJSONSchema(rb *RequestBody) (*Schema, bool) {
	if rb.Content == nil {
		return nil, false
	}
	mt, ok := rb.Content["application/json"]
	if !ok || mt.Schema == nil {
		return nil, false
	}
	return mt.Schema, true
}

// resolveSchema resolves $ref and allOf inline.
func resolveSchema(schema *Schema, index map[string]*Schema) *Schema {
	if schema == nil {
		return nil
	}

	if schema.Ref != "" {
		if resolved, ok := index[schema.Ref]; ok {
			return resolveSchema(resolved, index)
		}
	}

	if len(schema.AllOf) > 0 {
		merged := &Schema{
			Type:       "object",
			Properties: make(map[string]Schema),
		}
		for _, part := range schema.AllOf {
			resolved := resolveSchema(&part, index)
			if resolved == nil {
				continue
			}
			for k, v := range resolved.Properties {
				merged.Properties[k] = v
			}
			merged.Required = append(merged.Required, resolved.Required...)
		}
		return merged
	}

	if schema.Properties != nil {
		resolved := &Schema{
			Title:       schema.Title,
			Type:        schema.Type,
			Description: schema.Description,
			Properties:  make(map[string]Schema),
			Required:    schema.Required,
		}
		for k, v := range schema.Properties {
			resolved.Properties[k] = *resolveSchema(&v, index)
		}
		return resolved
	}

	return schema
}

// extractParams flattens top-level properties into ParamSpecs.
// Non-flaggable complex types (nested objects, arrays of objects) are filtered out.
func extractParams(schema *Schema) []ParamSpec {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var params []ParamSpec
	for name, prop := range schema.Properties {
		paramType, flaggable := mapJSONType(prop.Type, prop.Items)
		if !flaggable {
			continue
		}

		params = append(params, ParamSpec{
			Name:        name,
			Type:        paramType,
			Description: prop.Description,
			Required:    requiredSet[name],
			Default:     prop.Default,
			Enum:        prop.Enum,
			Minimum:     prop.Minimum,
		})
	}
	return params
}

// mapJSONType maps JSON Schema type to parameter type string.
// Returns empty string and false if the type is not flaggable.
func mapJSONType(t string, items *Schema) (string, bool) {
	switch t {
	case "string":
		return "string", true
	case "integer":
		return "integer", true
	case "number":
		return "number", true
	case "boolean":
		return "boolean", true
	case "array":
		if items != nil && items.Type == "string" {
			return "array", true
		}
		return "", false // arrays of objects not flaggable
	default:
		return "", false
	}
}

// hasProperties checks if a schema has any properties (for HasBody detection).
func hasProperties(schema *Schema) bool {
	return schema != nil && len(schema.Properties) > 0
}

// TagNameMap maps Chinese OpenAPI tags to English command names.
var TagNameMap = map[string]string{
	"探针管理":           "agent",
	"蜜罐管理":           "honeypot",
	"告警配置":           "alarm",
	"攻击者画像与威胁日志":     "event",
	"系统配置与系统信息":      "system",
	"Syslog 管理与系统日志": "syslog",
	"许可证信息":          "license",
	"用户操作日志":         "audit",
	"用户信息":           "account",
	"智学习":            "intellectual",
	"报告管理":           "report",
	"集中管理":           "captain",
	"日志归档管理":         "archive",
}

// CommandName derives the leaf command name from operationId and tag prefix.
func CommandName(opID, tagPrefix string) string {
	prefixes := []string{
		tagPrefix + "_",
		"list_", "get_", "update_", "delete_",
		"create_", "batch_", "set_", "apply_", "change_",
		"start_", "stop_", "reset_", "generate_", "confirm_",
		"download_", "modify_", "query_", "upload_",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(opID, p) {
			return opID[len(p):]
		}
	}
	return opID
}
