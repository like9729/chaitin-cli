package inspect

import (
	safelinecmd "github.com/chaitin/chaitin-cli/products/safeline/cmd"
	safelineruntime "github.com/chaitin/chaitin-cli/products/safeline/runtime"
	"github.com/spf13/cobra"
)

type result struct {
	OK        bool     `json:"ok"`
	Operation string   `json:"operation"`
	Warnings  []string `json:"warnings"`
	Errors    []string `json:"errors"`
	Data      data     `json:"data"`
}

type data struct {
	URL        string                               `json:"url"`
	Context    safelineruntime.Context              `json:"context"`
	SiteCreate safelineruntime.SiteCreateCapability `json:"site_create"`
}

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect SafeLine target version, mode, and CLI capabilities",
		RunE: func(c *cobra.Command, args []string) error {
			ctx, err := safelineruntime.ResolveContext(safelinecmd.NewClient(), safelineruntime.Options{
				VersionOverride:       safelinecmd.VersionOverride,
				OperationModeOverride: safelinecmd.OperationModeOverride,
				ConfigVersion:         safelinecmd.ConfigVersion,
				ConfigOperationMode:   safelinecmd.ConfigOperationMode,
			})
			if err != nil {
				return err
			}
			return safelinecmd.PrintResult(c, result{
				OK:        true,
				Operation: "inspect",
				Warnings:  ctx.Warnings,
				Errors:    []string{},
				Data: data{
					URL:        safelinecmd.URL,
					Context:    ctx,
					SiteCreate: safelineruntime.SiteCreateCapabilities(ctx),
				},
			})
		},
	}
}
