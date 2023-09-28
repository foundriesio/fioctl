package devices

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	deviceMine          bool
	deviceByTag         string
	deviceByTarget      string
	deviceByGroup       string
	deviceInactiveHours int
	deviceUuid          string
	showColumns         []string
	showPage            int
	paginationLimit     int
	paginationLimits    []int
)

type column struct {
	Formatter func(d *client.Device) string
}

func statusFormatter(d *client.Device) string {
	status := "OK"
	if len(d.Status) > 0 {
		status = d.Status
	}
	if !d.Online(deviceInactiveHours) {
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
	"name":           {func(d *client.Device) string { return d.Name }},
	"uuid":           {func(d *client.Device) string { return d.Uuid }},
	"factory":        {func(d *client.Device) string { return d.Factory }},
	"owner":          {ownerFormatter},
	"device-group":   {func(d *client.Device) string { return d.GroupName }},
	"target":         {func(d *client.Device) string { return d.TargetName }},
	"status":         {statusFormatter},
	"apps":           {func(d *client.Device) string { return strings.Join(d.DockerApps, ",") }},
	"up-to-date":     {func(d *client.Device) string { return fmt.Sprintf("%v", d.UpToDate) }},
	"tag":            {func(d *client.Device) string { return d.Tag }},
	"created-at":     {func(d *client.Device) string { return d.ChangeMeta.CreatedAt }},
	"created-by":     {func(d *client.Device) string { return d.ChangeMeta.CreatedBy }},
	"updated-at":     {func(d *client.Device) string { return d.ChangeMeta.UpdatedAt }},
	"updated-by":     {func(d *client.Device) string { return d.ChangeMeta.UpdatedBy }},
	"last-seen":      {func(d *client.Device) string { return d.LastSeen }},
	"ostree-hash":    {func(d *client.Device) string { return d.OstreeHash }},
	"current-update": {func(d *client.Device) string { return d.CurrentUpdate }},
	"is-prod":        {func(d *client.Device) string { return fmt.Sprintf("%v", d.IsProd) }},
	"is-wave":        {func(d *client.Device) string { return fmt.Sprintf("%v", d.IsWave) }},
}

func addPaginationFlags(cmd *cobra.Command) {
	paginationLimits = []int{10, 20, 30, 40, 50, 100, 200, 500, 1000}
	limitsStr := ""
	for i, limit := range paginationLimits {
		if i > 0 {
			limitsStr += ","
		}
		limitsStr += strconv.Itoa(limit)
	}

	cmd.Flags().IntVarP(&showPage, "page", "p", 1, "Page of devices to display when pagination is needed")
	cmd.Flags().IntVarP(&paginationLimit, "limit", "n", 500, "Number of devices to paginate by. Allowed values: "+limitsStr)
}

func addSortFlag(cmd *cobra.Command, flag, short, help string) {
	// Allows `--flag` to specify ascending sort order or a regular `--flag=asc`, `--flag=desc`.
	cmd.Flags().StringP(flag, short, "", help)
	cmd.Flags().Lookup(flag).NoOptDefVal = "asc"
}

func appendSortFlagValue(sortBy []string, cmd *cobra.Command, flag string, field string) []string {
	val, _ := cmd.Flags().GetString(flag)
	switch val {
	case "":
		break
	case "asc":
		sortBy = append(sortBy, field)
	case "desc":
		sortBy = append(sortBy, "-"+field)
	default:
		subcommands.DieNotNil(fmt.Errorf("Only 'asc, desc' values are allowed for %s but received: %s", flag, val))
	}
	return sortBy
}

func init() {
	var defCols = []string{
		"name", "target", "status", "apps", "up-to-date", "is-prod",
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
		Long:  "Available columns for display:\n\n  * " + strings.Join(allCols, "\n  * "),
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&deviceMine, "just-mine", "", false, "Only include devices owned by you")
	listCmd.Flags().StringVarP(&deviceByTag, "by-tag", "", "", "Only list devices configured with the given tag")
	listCmd.Flags().StringVarP(&deviceByTarget, "by-target", "", "", "Only list devices updated to the given target name")
	listCmd.Flags().StringVarP(&deviceByGroup, "by-group", "g", "", "Only list devices belonging to this group (factory is mandatory)")
	listCmd.Flags().IntVarP(&deviceInactiveHours, "offline-threshold", "", 4, "List the device as 'OFFLINE' if not seen in the last X hours")
	listCmd.Flags().StringVarP(&deviceUuid, "uuid", "", "", "Find device with the given UUID")
	listCmd.Flags().StringSliceVarP(&showColumns, "columns", "", defCols, "Specify which columns to display")
	addPaginationFlags(listCmd)
	addSortFlag(listCmd, "sort-by-name", "", "Sort by name (asc, desc); default sort is by owner and name")
	addSortFlag(listCmd, "sort-by-last-seen", "", "Sort by last-seen (asc, desc); default sort is by owner and name")
	listCmd.MarkFlagsMutuallyExclusive("sort-by-name", "sort-by-last-seen")
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

func assertPagination() {
	// hack until: https://github.com/spf13/pflag/issues/236
	for _, x := range paginationLimits {
		if x == paginationLimit {
			return
		}
	}
	subcommands.DieNotNil(fmt.Errorf("Invalid limit: %d", paginationLimit))
}

func showDeviceList(dl *client.DeviceList, showColumns []string) {
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
	t.Print()
	subcommands.ShowPages(showPage, dl.Next)
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing registered devices for: %s", factory)
	assertPagination()
	var sortBy []string
	sortBy = appendSortFlagValue(sortBy, cmd, "sort-by-last-seen", "last_seen")
	sortBy = appendSortFlagValue(sortBy, cmd, "sort-by-name", "name")

	name_ilike := ""
	if len(args) == 1 {
		name_ilike = sqlLikeIfy(args[0])
	}
	dl, err := api.DeviceList(
		deviceMine,
		deviceByTag,
		factory,
		deviceByGroup,
		name_ilike,
		deviceUuid,
		deviceByTarget,
		strings.Join(sortBy, ","),
		showPage,
		paginationLimit,
	)
	subcommands.DieNotNil(err)
	showDeviceList(dl, showColumns)
}
