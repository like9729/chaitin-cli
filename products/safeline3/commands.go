package safeline3

import "github.com/spf13/cobra"

func RegisterModules(cmd *cobra.Command) {
	cmd.AddCommand(newRawCommand())
	cmd.AddCommand(newNodeGroupCommand())
	cmd.AddCommand(newSiteCommand())
	cmd.AddCommand(newListenerCommand())
	cmd.AddCommand(newIPGroupCommand())
	cmd.AddCommand(newPolicyGroupCommand())
	cmd.AddCommand(newPolicyRuleCommand())
	cmd.AddCommand(newACLCommand())
	cmd.AddCommand(newLogCommand())
	cmd.AddCommand(newMonitorCommand())
	cmd.AddCommand(newSystemCommand())
	cmd.AddCommand(newNetworkCommand())
}
