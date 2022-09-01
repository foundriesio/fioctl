package el2g

import (
	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	devicesCmd := &cobra.Command{
		Use:   "devices",
		Short: "Manage devices for EdgeLock 2Go",
	}
	cmd.AddCommand(devicesCmd)

	devicesCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List devices configured to use EdgeLock 2Go",
		Run:   doList,
	})
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	devices, err := api.El2gDevices(factory)
	subcommands.DieNotNil(err)
	t := tabby.New()
	t.AddHeader("GROUP", "ID", "LAST CONNECTION")
	for _, device := range devices {
		t.AddLine(device.DeviceGroup, device.Id, device.LastConnection)
	}
	t.Print()
}
