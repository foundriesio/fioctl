package devices

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

var (
	deviceNoShared      bool
	deviceNoOwner       bool
	deviceByTag         string
	deviceByFactory     string
	deviceInactiveHours int
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list [pattern]",
		Short: "List devices registered to factories. Optionally include filepath style patterns to limit to device names. eg device-*",
		Run:   doList,
		Args:  cobra.MaximumNArgs(1),
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&deviceNoShared, "just-mine", "", false, "Only include devices owned by you")
	listCmd.Flags().BoolVarP(&deviceNoOwner, "skip-owner", "", false, "Do not include owner name when lising. (command will run faster)")
	listCmd.Flags().StringVarP(&deviceByTag, "by-tag", "", "", "Only list devices configured with the given tag")
	listCmd.Flags().StringVarP(&deviceByFactory, "by-factory", "", "", "Only list devices belonging to this factory")
	listCmd.Flags().IntVarP(&deviceInactiveHours, "offline-threshold", "", 4, "List the device as 'OFFLINE' if not seen in the last X hours")
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

func userName(factory, polisId string, cache map[string]string) string {
	name, ok := cache[polisId]
	if ok {
		return name
	}
	logrus.Debugf("Looking up user %s in factory %s", polisId, factory)
	users, err := api.UsersList(factory)
	if err != nil {
		logrus.Errorf("Unable to look up users: %s", err)
		return "???"
	}
	id := "<not in factory: " + polisId + ">"
	for _, user := range users {
		cache[user.PolisId] = user.Name
		if user.PolisId == polisId {
			id = user.Name
		}
	}
	return id
}

func doList(cmd *cobra.Command, args []string) {
	logrus.Debug("Listing registered devices")

	t := tabby.New()
	if deviceNoOwner {
		t.AddHeader("NAME", "FACTORY", "TARGET", "STATUS", "APPS", "UP TO DATE")
	} else {
		t.AddHeader("NAME", "FACTORY", "OWNER", "TARGET", "STATUS", "APPS", "UP TO DATE")
	}

	cache := make(map[string]string)

	var dl *client.DeviceList
	for {
		var err error
		if dl == nil {
			name_ilike := ""
			if len(args) == 1 {
				name_ilike = sqlLikeIfy(args[0])
			}
			if len(deviceByFactory) > 0 {
				deviceNoShared = true
			}
			dl, err = api.DeviceList(!deviceNoShared, deviceByTag, deviceByFactory, name_ilike)
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
			if deviceNoOwner {
				t.AddLine(device.Name, device.Factory, device.TargetName, status, strings.Join(device.DockerApps, ","), device.UpToDate)
			} else {
				owner := userName(device.Factory, device.Owner, cache)
				t.AddLine(device.Name, device.Factory, owner, device.TargetName, status, strings.Join(device.DockerApps, ","), device.UpToDate)
			}
		}
	}
	t.Print()
}
