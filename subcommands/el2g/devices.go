package el2g

import (
	"fmt"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var add string

func init() {
	devicesCmd := &cobra.Command{
		Use:   "devices [<device-id>]",
		Short: "List devices in EdgeLock2Go",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				doDevices(cmd, args)
			} else {
				doDevice(cmd, args)
			}
		},
		Example: `
# List all devices
fioctl el2g devices 

# Show details of a device
fioctl el2g devices <device-id>

# Add a device with an se050
fioctl el2g devices --add 935389312472 <device-id>
`,
	}
	devicesCmd.Flags().StringVarP(&add, "add", "", "", "Whitelist the device for the given nc12 product id")
	cmd.AddCommand(devicesCmd)
}

func doDevices(cmd *cobra.Command, args []string) {
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

func doDevice(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	deviceId := args[0]

	if len(add) > 0 {
		subcommands.DieNotNil(api.El2gAddDevice(factory, add, deviceId))
		return
	}

	info, err := api.El2gProductInfo(factory, deviceId)
	subcommands.DieNotNil(err)
	fmt.Println("Hardware Type:", info.Type)
	fmt.Println("Hardware 12NC:", info.Nc12)

	fmt.Println("Secure Objects:")
	objects, err := api.El2gSecureObjectProvisionings(factory, deviceId)
	subcommands.DieNotNil(err)
	t := subcommands.Tabby(1, "NAME", "TYPE", "STATUS")
	for _, obj := range objects {
		t.AddLine(obj.Name, obj.Type, obj.State)
	}
	t.Print()
}
