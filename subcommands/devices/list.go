package devices

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	deviceNoShared      bool
	deviceByTag         string
	deviceByFactory     string
	deviceByGroup       string
	deviceInactiveHours int
	deviceUuid          string
	showColumns         []string
)

type column struct {
	Formatter func(d *client.Device) string
}

func statusFormatter(d *client.Device) string {
	status := "OK"
	if len(d.Status) > 0 {
		status = d.Status
	}
	if len(d.LastSeen) > 0 && !d.Online(deviceInactiveHours) {
		status = "OFFLINE"
	}
	return status
}

var ownerCache = make(map[string]string)

func ownerFormatter(d *client.Device) string {
	name, ok := ownerCache[d.Owner]
	if ok {
		return name
	}
	logrus.Debugf("Looking up user %s in factory %s", d.Owner, d.Factory)
	users, err := api.UsersList(d.Factory)
	if err != nil {
		logrus.Errorf("Unable to look up users: %s", err)
		return "???"
	}
	id := "<not in factory: " + d.Factory + ">"
	for _, user := range users {
		ownerCache[user.PolisId] = user.Name
		if user.PolisId == d.Owner {
			id = user.Name
		}
	}
	return id
}

var Columns = map[string]column{
	"name":          {func(d *client.Device) string { return d.Name }},
	"uuid":          {func(d *client.Device) string { return d.Uuid }},
	"factory":       {func(d *client.Device) string { return d.Factory }},
	"owner":         {ownerFormatter},
	"target":        {func(d *client.Device) string { return d.TargetName }},
	"status":        {statusFormatter},
	"apps":          {func(d *client.Device) string { return strings.Join(d.DockerApps, ",") }},
	"up-to-date":    {func(d *client.Device) string { return fmt.Sprintf("%v", d.UpToDate) }},
	"tags":          {func(d *client.Device) string { return strings.Join(d.Tags, ",") }},
	"created-at":    {func(d *client.Device) string { return d.CreatedAt }},
	"last-seen":     {func(d *client.Device) string { return d.LastSeen }},
	"ostree-hash":   {func(d *client.Device) string { return d.OstreeHash }},
	"curent-update": {func(d *client.Device) string { return d.CurrentUpdate }},
	"is-prod":       {func(d *client.Device) string { return fmt.Sprintf("%v", d.IsProd) }},
	"is-wave":       {func(d *client.Device) string { return fmt.Sprintf("%v", d.IsWave) }},
}

func init() {
	var defCols = []string{
		"name", "factory", "target", "status", "apps", "up-to-date", "is-prod",
	}

	allCols := make([]string, 0, len(Columns))
	for k := range Columns {
		allCols = append(allCols, k)
	}
	sort.Strings(allCols)
	listCmd := &cobra.Command{
		Use:   "list [pattern]",
		Short: "List devices registered to factories. Optionally include filepath style patterns to limit to device names. eg device-*",
		Run:   doList,
		Args:  cobra.MaximumNArgs(1),
		Long:  "Available columns for display:\n  * " + strings.Join(allCols, "\n  * "),
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&deviceNoShared, "just-mine", "", false, "Only include devices owned by you")
	listCmd.Flags().StringVarP(&deviceByTag, "by-tag", "", "", "Only list devices configured with the given tag")
	listCmd.Flags().StringVarP(&deviceByFactory, "by-factory", "f", "", "Only list devices belonging to this factory")
	listCmd.Flags().StringVarP(&deviceByGroup, "by-group", "g", "", "Only list devices belonging to this group (factory is mandatory)")
	listCmd.Flags().IntVarP(&deviceInactiveHours, "offline-threshold", "", 4, "List the device as 'OFFLINE' if not seen in the last X hours")
	listCmd.Flags().StringVarP(&deviceUuid, "uuid", "", "", "Find device with the given UUID")
	listCmd.Flags().StringSliceVarP(&showColumns, "columns", "", defCols, "Specify which columns to display")
}

// We allow pattern matching using filepath.Match type * and ?
// ie * matches everything and ? matches a single character.
// In sql we need * and ? to maps to % and _ respectively
// Since _ is a special character we need to escape that. And
//
// Soo... a pattern like: H?st_b* would become: H_st\_b%
// and would match stuff like host_b and hast_FOOO
func sqlLikeIfy(filePathLike string) string {
	// %25 = urlencode("%")
	sql := strings.Replace(filePathLike, "*", "%25", -1)
	sql = strings.Replace(sql, "_", "\\_", -1)
	sql = strings.Replace(sql, "?", "_", -1)
	logrus.Debugf("Converted query(%s) -> %s", filePathLike, sql)
	return sql
}

func doList(cmd *cobra.Command, args []string) {
	logrus.Debug("Listing registered devices")

	t := tabby.New()
	var cols = make([]interface{}, len(showColumns))
	for idx, c := range showColumns {
		if _, ok := Columns[c]; !ok {
			fmt.Println("ERROR: Invalid column name:", c)
			os.Exit(1)
		}
		cols[idx] = strings.ToUpper(c)
	}
	t.AddHeader(cols...)

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
			} else if len(deviceByGroup) > 0 {
				subcommands.DieNotNil(fmt.Errorf("A factory is mandatory to filter by group"))
			}
			dl, err = api.DeviceList(
				!deviceNoShared, deviceByTag, deviceByFactory, deviceByGroup, name_ilike, deviceUuid)
		} else {
			if dl.Next != nil {
				dl, err = api.DeviceListCont(*dl.Next)
			} else {
				break
			}
		}
		subcommands.DieNotNil(err)
		row := make([]interface{}, len(showColumns))
		for _, device := range dl.Devices {
			if len(device.TargetName) == 0 {
				device.TargetName = "???"
			}
			for idx, col := range showColumns {
				col := Columns[col]
				row[idx] = col.Formatter(&device)
			}
			t.AddLine(row...)
		}
	}
	t.Print()
}
