package events

import (
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	api *client.Api
)

var cmd = &cobra.Command{
	Use:   "event-queues",
	Short: "Manage event queues configured for a Factory",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
	Long: `Event queues provide a way for customers to receive notifications about events
happening in a Factory such as when a device is first seen or has started an
over-the-air update.

There are two types of event queues: push and pull. A pull queue works like a
traditional message queue system. Push queues are synonymous with web hooks.`,
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)
	return cmd
}
