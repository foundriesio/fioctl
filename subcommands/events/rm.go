package events

import (
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "rm <label>",
		Short: "Remove an event queue",
		Args:  cobra.ExactArgs(1),
		Run:   doRemove,
	})
}

func doRemove(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Removing event queue for: %s", factory)

	err := api.EventQueuesDelete(factory, args[0])
	subcommands.DieNotNil(err)
}
