package el2g

import (
	"errors"
	"fmt"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	add string
	del string
)

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

# Add a device with an se050 (product ID: 935389312472)
# Device ID can be found on a device by running:
#  $ ssscli se05x uid | grep "Unique ID:" | cut -d: -f2
#  ssscli se05x uid | grep "Unique ID:" | cut -d: -f2
#  04005001eee3ba1ee96e60047e57da0f6880
# This ID is hexadecimal and must be prefixed in the CLI with 0x. For example:
fioctl el2g devices --add 935389312472 0x04005001eee3ba1ee96e60047e57da0f6880

# Delete a device with an se050
fioctl el2g devices --del 935389312472 <device-id>
`,
	}
	devicesCmd.Flags().StringVarP(&add, "add", "", "", "Whitelist the device for the given nc12 product id")
	devicesCmd.Flags().StringVarP(&del, "del", "", "", "Unclaim device for the given nc12 product id")
	cmd.AddCommand(devicesCmd)
}

func doDevices(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	if len(add) > 0 || len(del) > 0 {
		cmd.Usage()
		return
	}

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

	if len(add) > 0 && len(del) > 0 {
		subcommands.DieNotNil(errors.New("--add and --del are mutually exclusive"))
	} else if len(add) > 0 {
		subcommands.DieNotNil(api.El2gAddDevice(factory, add, deviceId))
		return
	} else if len(del) > 0 {
		subcommands.DieNotNil(api.El2gDelDevice(factory, del, deviceId))
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
