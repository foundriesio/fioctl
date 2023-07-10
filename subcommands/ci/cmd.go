package ci

import (
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

var api *client.Api

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Commands that interact with the FoundriesFactory CI server",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = subcommands.Login(cmd)
		},
	}
	return cmd
}
