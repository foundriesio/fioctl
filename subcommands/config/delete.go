package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <file>",
		Short: "Delete file from the current configuration",
		Run:   doConfigDelete,
		Args:  cobra.ExactArgs(1),
	})
}

func doConfigDelete(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Deleting file from config for %s", factory)

	subcommands.DieNotNil(api.FactoryDeleteConfig(factory, args[0]))
}
