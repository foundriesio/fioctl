package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	configCmd.AddCommand(&cobra.Command{
		Use:   "delete <device> <file>",
		Short: "Delete file from the current configuration",
		Run:   doConfigDelete,
		Args:  cobra.ExactArgs(2),
	})
}

func doConfigDelete(cmd *cobra.Command, args []string) {
	logrus.Debug("Deleting file from device config")

	d := getDeviceApi(cmd, args[0])
	subcommands.DieNotNil(d.DeleteConfig(args[1]))
}
