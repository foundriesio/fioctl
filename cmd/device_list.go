package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var deviceListCmd = &cobra.Command{
	Use:   "list [pattern]",
	Short: "List devices registered to factories. Optionally include filepath style patterns to limit to device names. eg device-*",
	Run:   doDeviceList,
	Args:  cobra.MinimumNArgs(0),
}
var (
	deviceNoShared bool
	deviceByTag    string
)

func init() {
	deviceCmd.AddCommand(deviceListCmd)
	deviceListCmd.Flags().BoolVarP(&deviceNoShared, "just-mine", "", false, "Only include devices owned by you")
	deviceListCmd.Flags().StringVarP(&deviceByTag, "by-tag", "", "", "Only list devices configured with the given tag")
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
			if len(args) > 0 {
				match := false
				for _, arg := range args {
					if match, _ = filepath.Match(arg, device.Name); match == true {
						break
					}
				}
				if !match {
					logrus.Debugf("Device(%v) does not match: %s", device, args)
					continue
				}
			}
			if len(deviceByTag) > 0 && !intersectionInSlices([]string{deviceByTag}, device.Tags) {
				logrus.Debugf("Device(%v) does not include tag", device)
				continue
			}
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
			if len(device.Tags) > 0 {
				fmt.Printf("\tTags:\t\t%s\n", strings.Join(device.Tags, ","))
			}
			if len(device.DockerApps) > 0 {
				fmt.Printf("\tDocker Apps:\t%s\n", strings.Join(device.DockerApps, ","))
			}
			if len(device.Status) > 0 {
				fmt.Printf("\tStatus:\t\t%s\n", device.Status)
			}
			if len(device.CurrentUpdate) > 0 {
				fmt.Printf("\tUpdate Id:\t%s\n", device.CurrentUpdate)
			}
		}
	}
}
