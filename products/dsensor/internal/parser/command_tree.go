package parser

import (
	"github.com/spf13/cobra"
)

// Runner is the function called by each leaf command to execute an API call.
// It receives the operation spec and a map of set flag values.
type Runner func(spec CommandSpec, body []byte) ([]byte, error)

// DefaultRunner is set by the CLI entry point after creating the HTTP client.
var DefaultRunner Runner

// BuildCommandTree creates a cobra command tree from parsed CommandSpecs.
// Returns a root command with tag-grouped parent commands and leaf subcommands.
func BuildCommandTree(cmds []CommandSpec) *cobra.Command {
	tagGroups := make(map[string][]CommandSpec)
	for _, cmd := range cmds {
		for _, tag := range cmd.Tags {
			tagGroups[tag] = append(tagGroups[tag], cmd)
		}
	}

	root := &cobra.Command{
		Use:   "dsensor",
		Short: "谛听 API CLI 工具",
	}

	// Sort tags by mapped name for consistent output
	type tagEntry struct {
		chinese string
		english string
		cmds    []CommandSpec
	}
	var entries []tagEntry
	for chinese, groupCmds := range tagGroups {
		english, ok := TagNameMap[chinese]
		if !ok {
			continue
		}
		entries = append(entries, tagEntry{chinese: chinese, english: english, cmds: groupCmds})
	}
	// Sort by English name (simple insertion order by map iteration is not guaranteed)
	// We sort alphabetically here

	for _, entry := range entries {
		parent := &cobra.Command{
			Use:   entry.english,
			Short: entry.chinese,
		}
		root.AddCommand(parent)

		// Track used command names to avoid duplicates
		usedNames := make(map[string]int)

		for _, cmd := range entry.cmds {
			name := CommandName(cmd.OperationID, entry.english)

			// Handle duplicate names by appending operationId suffix
			if count, exists := usedNames[name]; exists {
				usedNames[name] = count + 1
				name = name + "-" + cmd.OperationID
			} else {
				usedNames[name] = 1
			}

			leaf := &cobra.Command{
				Use:   name,
				Short: cmd.Summary,
				Long:  buildLongDesc(cmd),
				RunE:  makeRunE(cmd),
			}

			// Add field flags from body params
			addFieldFlags(leaf, cmd.BodyParams)

			// Add --body and --body-file for all commands that have a body
			if cmd.HasBody {
				leaf.Flags().String("body", "", "请求体 JSON 字符串")
				leaf.Flags().String("body-file", "", "从文件读取请求体 JSON")
			}

			parent.AddCommand(leaf)
		}
	}

	return root
}

func buildLongDesc(cmd CommandSpec) string {
	desc := cmd.Summary
	if cmd.Path != "" {
		desc += "\n\n路径: " + cmd.Method + " " + cmd.Path
	}
	if len(cmd.BodyParams) > 0 {
		desc += "\n\n参数:"
		for _, p := range cmd.BodyParams {
			req := ""
			if p.Required {
				req = " (必填)"
			}
			desc += "\n  --" + toFlagName(p.Name) + "  " + p.Type + req
			if p.Description != "" {
				desc += "  " + p.Description
			}
		}
	}
	return desc
}

func makeRunE(cmd CommandSpec) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		bodyFlag, _ := c.Flags().GetString("body")
		bodyFile, _ := c.Flags().GetString("body-file")

		hasFieldFlags := anyFieldFlagSet(c, cmd.BodyParams)

		if (bodyFlag != "" || bodyFile != "") && hasFieldFlags {
			return ErrMutualExclusive
		}

		var body []byte
		if bodyFlag != "" {
			body = []byte(bodyFlag)
		} else if bodyFile != "" {
			var err error
			body, err = readFile(bodyFile)
			if err != nil {
				return err
			}
		} else if hasFieldFlags {
			body = buildBodyFromFlags(c, cmd.BodyParams)
		} else {
			body = []byte("{}")
		}

		if DefaultRunner == nil {
			return ErrNoRunner
		}

		resp, err := DefaultRunner(cmd, body)
		if err != nil {
			return err
		}

		// Print response to stdout
		c.Println(string(resp))
		return nil
	}
}
