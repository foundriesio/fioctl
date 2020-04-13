package secrets

import (
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var api *client.Api

var cmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secret crendentials configured in a factory",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)
	return cmd
}
