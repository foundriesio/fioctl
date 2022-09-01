package el2g

import (
	"fmt"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var production bool

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

	devicesCmd.AddCommand(&cobra.Command{
		Use:   "show <device-id>",
		Short: "Show the integrations details for a device",
		Args:  cobra.ExactArgs(1),
		Run:   doShow,
	})

	add := &cobra.Command{
		Use:   "add <NC12 product-id> <device-id>",
		Short: "Grant device access to EdgeLock 2GO",
		Args:  cobra.ExactArgs(2),
		Run:   doAdd,
		Example: `# Add a device with an SE050 (product ID: 935389312472)
# The product IDs configured for you factory can be found by running
#  fioctl el2g status
# Device ID can be found on a device by running:
#  $ ssscli se05x uid | grep "Unique ID:" | cut -d: -f2
#  ssscli se05x uid | grep "Unique ID:" | cut -d: -f2
#  04005001eee3ba1ee96e60047e57da0f6880
# This ID is hexadecimal and must be prefixed in the CLI with 0x. For example:
fioctl el2g devices add 935389312472 0x04005001eee3ba1ee96e60047e57da0f6880

# Add a production device with an SE051 HSM (product ID: 935414457472)
fioctl el2g devices add --production 935414457472 0x04005001eee3ba1ee96e60047e57da0f6880
`,
	}
	add.Flags().BoolVarP(&production, "production", "", false, "A production device")
	devicesCmd.AddCommand(add)
	del := &cobra.Command{
		Use:   "delete <NC12 product-id> <device-id>",
		Short: "Revoke device access to EdgeLock 2GO",
		Args:  cobra.ExactArgs(2),
		Run:   doDelete,
	}
	del.Flags().BoolVarP(&production, "production", "", false, "A production device")
	devicesCmd.AddCommand(del)
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

func doShow(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	deviceId := args[0]

	info, err := api.El2gProductInfo(factory, deviceId)
	subcommands.DieNotNil(err)
	fmt.Println("Hardware Type:", info.Type)
	fmt.Println("Hardware 12NC:", info.Nc12)

	fmt.Println("Secure Objects:")
	objects, err := api.El2gSecureObjectProvisionings(factory, deviceId)
	subcommands.DieNotNil(err)
	t := subcommands.Tabby(1, "NAME", "TYPE", "STATUS")
	foundCert := false
	for _, obj := range objects {
		t.AddLine(obj.Name, obj.Type, obj.State)
		if len(obj.Cert) > 0 {
			foundCert = true
		}
	}
	t.Print()

	if foundCert {
		fmt.Println("Certificates:")
		for _, obj := range objects {
			if len(obj.Cert) > 0 {
				fmt.Println("#", obj.Name)
				fmt.Println(obj.Cert)
				fmt.Println()
			}
		}
	}

}

func doAdd(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	prodId := args[0]
	deviceId := args[1]
	subcommands.DieNotNil(api.El2gAddDevice(factory, prodId, deviceId, production))
}

func doDelete(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	prodId := args[0]
	deviceId := args[1]
	subcommands.DieNotNil(api.El2gDeleteDevice(factory, prodId, deviceId, production))
}
