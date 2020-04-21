package devices

import (
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	api       *client.Api
	listLimit int
)

var cmd = &cobra.Command{
	Use:     "devices",
	Aliases: []string{"device"},
	Short:   "Manage devices registered to a factory",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Device configuration",
}

var updatesCmd = &cobra.Command{
	Use:   "updates",
	Short: "Device update history",
}

func NewCommand() *cobra.Command {
	cmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
	cmd.AddCommand(configCmd)
	cmd.AddCommand(updatesCmd)
	return cmd
}
