package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list-denied",
		Short: "List device UUIDs that have been denied access to the device-gateway",
		Run:   doListDenied,
		Long: `Devices created using a factory-registration-reference server get created
on-demand. Because of this, devices are placed into a deny-list when
they are deleted, so that they can't continue to access the system by getting 
re-created.`,
	}
	cmd.AddCommand(listCmd)
	addPaginationFlags(listCmd)
}

func doListDenied(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing denied devices for: %s", factory)
	assertPagination()

	dl, err := api.DeviceListDenied(factory, showPage, paginationLimit)
	subcommands.DieNotNil(err)
	showDeviceList(dl, []string{"uuid", "name", "owner"})
}
