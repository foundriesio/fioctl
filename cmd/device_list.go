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
	Use:   "list [pattern]",
	Short: "List devices registered to factories. Optionally include filepath style patterns to limit to device names. eg device-*",
	Run:   doDeviceList,
	Args:  cobra.MaximumNArgs(1),
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

// We allow pattern matching using filepath.Match type * and ?
// ie * matches everything and ? matches a single character.
// In sql we need * and ? to maps to % and _ respectively
// Since _ is a special character we need to escape that. And
//
// Soo... a pattern like: H?st_b* would become: H_st\_b%
// and would match stuff like host_b and hast_FOOO
func sqlLikeIfy(filePathLike string) string {
	sql := strings.Replace(filePathLike, "*", "%", -1)
	sql = strings.Replace(sql, "_", "\\_", -1)
	sql = strings.Replace(sql, "?", "_", -1)
	logrus.Debugf("Converted query(%s) -> %s", filePathLike, sql)
	return sql
}

func doDeviceList(cmd *cobra.Command, args []string) {
	logrus.Debug("Listing registered devices")

	var dl *client.DeviceList
	for {
		var err error
		if dl == nil {
			name_ilike := ""
			if len(args) == 1 {
				name_ilike = sqlLikeIfy(args[0])
			}
			dl, err = api.DeviceList(!deviceNoShared, deviceByTag, name_ilike)
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
