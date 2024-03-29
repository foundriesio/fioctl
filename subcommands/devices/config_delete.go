package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	factory := viper.GetString("factory")
	logrus.Debug("Deleting file from device config")

	subcommands.DieNotNil(api.DeviceDeleteConfig(factory, args[0], args[1]))
}
