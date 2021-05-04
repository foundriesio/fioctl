package devices

import (
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "chown <device> <new-owner-id>",
		Short: "Change the device's owner",
		Run:   doChown,
		Args:  cobra.ExactArgs(2),
		Long: `Change the owner of a device. This command can only be run by factory admins 
and owners. The new owner-id can be found by running 'fioctl users'`,
	})
}

func doChown(cmd *cobra.Command, args []string) {
	logrus.Debug("Chown %r", args)
	device := args[0]
	owner := args[1]

	subcommands.DieNotNil(api.DeviceChown(device, owner))
}
