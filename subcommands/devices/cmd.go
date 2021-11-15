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
	Use:   "updates <device> [<update-id>]",
	Short: "Show updates performed on a device",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			doListUpdates(cmd, args)
		} else {
			doShowUpdate(cmd, args)
		}
	},
	Example: `
# List all updates performed on a device:
fioctl devices updates <device1>

# List the last 2 updates:
fioctl devices updates <device> -n2

# Show the details of an update:
fioctl devices updates <device> <update-id>

# Show the most recent update with bash help:
fioctl devices updates <device> $(fioctl devices updates <device> -n1 | tail -n1 | cut -f1 -d\ )
`,
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)

	updatesCmd.Flags().IntVarP(&listLimit, "limit", "n", 0, "Limit the number of updates displayed.")

	cmd.AddCommand(configCmd)
	cmd.AddCommand(updatesCmd)
	return cmd
}
