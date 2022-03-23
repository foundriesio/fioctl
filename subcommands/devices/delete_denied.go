package devices

import (
	"fmt"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "delete-denied <uuid> [<uuid>...]",
		Short: "Remove a device UUID from the deny list",
		Run:   doDeleteDenied,
		Args:  cobra.MinimumNArgs(1),
		Long: `Remove a device UUID from the deny list so that the UUID can be re-used.
This is handy for Factories using HSMs and a factory-registration-reference
server.`,
	})
}

func doDeleteDenied(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debug("Deleting %r", args)

	for _, uuid := range args {
		fmt.Printf("Deleting %s .. ", uuid)
		subcommands.DieNotNil(api.DeviceDeleteDenied(factory, uuid))
		fmt.Println("ok")
	}
}
