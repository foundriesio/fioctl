package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
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
	deviceNoShared      bool
	deviceByTag         string
	deviceInactiveHours int
)

func init() {
	deviceCmd.AddCommand(deviceListCmd)
	deviceListCmd.Flags().BoolVarP(&deviceNoShared, "just-mine", "", false, "Only include devices owned by you")
	deviceListCmd.Flags().StringVarP(&deviceByTag, "by-tag", "", "", "Only list devices configured with the given tag")
	deviceListCmd.Flags().IntVarP(&deviceInactiveHours, "offline-threshold", "", 4, "List the device as 'OFFLINE' if not seen in the last X hours")
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

	t := tabby.New()
	t.AddHeader("NAME", "FACTORY", "TARGET", "STATUS", "APPS")
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
			if len(device.TargetName) == 0 {
				device.TargetName = "???"
			}
			status := "OK"
			if len(device.Status) > 0 {
				status = device.Status
			}
			if len(device.LastSeen) > 0 {
				t, err := time.Parse("2006-01-02T15:04:05", device.LastSeen)
				if err == nil {
					duration := time.Now().Sub(t)
					if duration.Hours() > float64(deviceInactiveHours) {
						status = "OFFLINE"
					}
				} else {
					logrus.Error(err)
				}
			}
			t.AddLine(device.Name, device.Factory, device.TargetName, status, strings.Join(device.DockerApps, ","))
		}
	}
	t.Print()
}
