package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	ErrNoRunner        = errors.New("no API runner configured (missing --url?)")
	ErrMutualExclusive = errors.New("--body/--body-file 与字段级 flag 互斥，不能同时使用")
)

func toFlagName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func addFieldFlags(cmd *cobra.Command, params []ParamSpec) {
	for _, p := range params {
		flagName := toFlagName(p.Name)
		desc := p.Description
		if len(p.Enum) > 0 {
			desc += fmt.Sprintf(" (可选值: %v)", p.Enum)
		}
		if p.Default != nil {
			desc += fmt.Sprintf(" (默认: %v)", p.Default)
		}

		switch p.Type {
		case "string":
			defVal := ""
			if p.Default != nil {
				defVal = fmt.Sprintf("%v", p.Default)
			}
			cmd.Flags().String(flagName, defVal, desc)
		case "integer":
			defVal := 0
			if p.Default != nil {
				switch v := p.Default.(type) {
				case float64:
					defVal = int(v)
				case int:
					defVal = v
				}
			}
			cmd.Flags().Int(flagName, defVal, desc)
		case "number":
			defVal := 0.0
			if p.Default != nil {
				switch v := p.Default.(type) {
				case float64:
					defVal = v
				}
			}
			cmd.Flags().Float64(flagName, defVal, desc)
		case "boolean":
			defVal := false
			if p.Default != nil {
				switch v := p.Default.(type) {
				case bool:
					defVal = v
				}
			}
			cmd.Flags().Bool(flagName, defVal, desc)
		case "array":
			cmd.Flags().StringSlice(flagName, nil, desc)
		}

		if p.Required {
			_ = cmd.MarkFlagRequired(flagName)
		}
	}
}

func anyFieldFlagSet(cmd *cobra.Command, params []ParamSpec) bool {
	for _, p := range params {
		flagName := toFlagName(p.Name)
		if cmd.Flags().Changed(flagName) {
			return true
		}
	}
	return false
}

func buildBodyFromFlags(cmd *cobra.Command, params []ParamSpec) []byte {
	body := make(map[string]any)
	for _, p := range params {
		flagName := toFlagName(p.Name)
		if !cmd.Flags().Changed(flagName) {
			if p.Default != nil {
				body[p.Name] = p.Default
			}
			continue
		}
		switch p.Type {
		case "string":
			val, _ := cmd.Flags().GetString(flagName)
			body[p.Name] = val
		case "integer":
			val, _ := cmd.Flags().GetInt(flagName)
			body[p.Name] = val
		case "number":
			val, _ := cmd.Flags().GetFloat64(flagName)
			body[p.Name] = val
		case "boolean":
			val, _ := cmd.Flags().GetBool(flagName)
			body[p.Name] = val
		case "array":
			val, _ := cmd.Flags().GetStringSlice(flagName)
			body[p.Name] = val
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	// Validate JSON
	var tmp any
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, fmt.Errorf("文件不是合法 JSON: %w", err)
	}
	return data, nil
}
