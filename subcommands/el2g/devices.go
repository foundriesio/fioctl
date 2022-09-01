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
