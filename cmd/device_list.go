package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List devices registered to factories.",
	Run:   doDeviceList,
}
var deviceNoShared bool

func init() {
	deviceCmd.AddCommand(deviceListCmd)
	deviceListCmd.Flags().BoolVarP(&deviceNoShared, "just-mine", "", false, "Only include devices owned by you")
}

func doDeviceList(cmd *cobra.Command, args []string) {
	logrus.Debug("Listing registered devices")

	var dl *client.DeviceList
	for {
		var err error
		if dl == nil {
			dl, err = api.DeviceList(!deviceNoShared)
		} else {
			if dl.Next != nil {
				dl, err = api.DeviceListCont(*dl.Next)
			} else {
				break
			}
		}
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		for _, device := range dl.Devices {
			fmt.Printf("= %s", device.Name)
			if device.Network != nil {
				fmt.Printf("\tHostname(%s) IPv4(%s) MAC(%s)\n", device.Network.Hostname, device.Network.Ipv4, device.Network.MAC)
			} else {
				fmt.Printf("\n")
			}
			fmt.Printf("\tUUID:\t\t%s\n", device.Uuid)
			fmt.Printf("\tOwner:\t\t%s\n", device.Owner)
			fmt.Printf("\tFactory:\t%s\n", device.Factory)
			fmt.Printf("\tTarget:\t\t%s / sha256(%s)\n", device.TargetName, device.OstreeHash)
			fmt.Printf("\tOstree Hash:\t%s\n", device.OstreeHash)
			fmt.Printf("\tCreated:\t%s\n", device.CreatedAt)
			fmt.Printf("\tLast Seen:\t%s\n", device.LastSeen)
			if len(device.DockerApps) > 0 {
				fmt.Printf("\tDocker Apps:\t%s\n", strings.Join(device.DockerApps, ","))
			}
		}
	}
}
