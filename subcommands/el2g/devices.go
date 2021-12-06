package el2g

import (
	"fmt"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	devicesCmd := &cobra.Command{
		Use:   "devices",
		Short: "List devices in EdgeLock2Go",
		Run:   doDevices,
	}
	cmd.AddCommand(devicesCmd)
}

func doDevices(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	devices, err := api.El2gDevices(factory)
	subcommands.DieNotNil(err)
	fmt.Println("Devices")
	t := tabby.New()
	t.AddHeader("GROUP", "ID", "LAST CONNECTION")
	for _, device := range devices {
		t.AddLine(device.DeviceGroup, device.Id, device.LastConnection)
	}
	t.Print()
}
