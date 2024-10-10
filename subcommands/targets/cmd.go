package targets

import (
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var api *client.Api

var cmd = &cobra.Command{
	Use:   "targets",
	Short: "Manage Factory TUF targets",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)
	return cmd
}
