package devices

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	Short:   "Manage devices registered to a Factory",
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

	addUuidFlagToChildren(cmd)

	return cmd
}

func addUuidFlagToChildren(c *cobra.Command) {
	ignores := []string{"list-denied", "list", "delete-denied"}
	for _, child := range c.Commands() {
		if child.HasSubCommands() {
			addUuidFlagToChildren(child)
		} else if !slices.Contains(ignores, child.Name()) {
			child.Flags().BoolP("by-uuid", "u", false, "Look up device by UUID rather than name")
		}
	}
}

func getDeviceApi(cmd *cobra.Command, name string) client.DeviceApi {
	byUuid, err := cmd.Flags().GetBool("by-uuid")
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	if byUuid && err == nil {
		return api.DeviceApiByUuid(viper.GetString("factory"), name)
	}
	return api.DeviceApiByName(viper.GetString("factory"), name)
}

func getDevice(cmd *cobra.Command, name string) *client.Device {
	dapi := getDeviceApi(cmd, name)
	d, err := dapi.Get()
	subcommands.DieNotNil(err)
	return d
}
