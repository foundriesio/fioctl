package targets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	listProd    bool
	listRaw     bool
	listByTag   string
	showColumns []string
)

// Represents the details we use for displaying a single OTA "build"
type targetListing struct {
	version      int
	hardwareIds  []string
	tags         []string
	apps         []string
	origin       string
	manifestSha  string
	overridesSha string
	containerSha string
}

type byTargetKey []string

func (t byTargetKey) Len() int {
	return len(t)
}
func (t byTargetKey) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
func (t byTargetKey) Less(i, j int) bool {
	var tagsI, tagsJ string
	var verI, verJ int
	fmt.Sscanf(t[i], "%d-%s", &verI, &tagsI)
	fmt.Sscanf(t[j], "%d-%s", &verJ, &tagsJ)
	if verI == verJ {
		return tagsI < tagsJ
	}
	return verI < verJ
}

type column struct {
	Formatter func(tl *targetListing) string
}

var Columns = map[string]column{
	"version":        {func(tl *targetListing) string { return strconv.Itoa(tl.version) }},
	"tags":           {func(tl *targetListing) string { return strings.Join(tl.tags, ",") }},
	"apps":           {func(tl *targetListing) string { return strings.Join(tl.apps, ",") }},
	"hardware-ids":   {func(tl *targetListing) string { return strings.Join(tl.hardwareIds, ",") }},
	"origin":         {func(tl *targetListing) string { return tl.origin }},
	"manifest-sha":   {func(tl *targetListing) string { return tl.manifestSha }},
	"overrides-sha":  {func(tl *targetListing) string { return tl.overridesSha }},
	"containers-sha": {func(tl *targetListing) string { return tl.containerSha }},
}

func init() {
	var defCols = []string{"version", "tags", "apps", "origin"}
	allCols := make([]string, 0, len(Columns))
	for k := range Columns {
		allCols = append(allCols, k)
	}
	sort.Strings(allCols)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List targets.",
		Run:   doList,
		Long:  "Available columns for display:\n\n  * " + strings.Join(allCols, "\n  * "),
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listRaw, "raw", "r", false, "Print raw targets.json")
	listCmd.Flags().BoolVarP(&listProd, "production", "", false, "Show the production version targets.json")
	listCmd.Flags().StringVarP(&listByTag, "by-tag", "", "", "Only list targets that match the given tag")
	listCmd.Flags().StringSliceVarP(&showColumns, "columns", "", defCols, "Specify which columns to display")
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing targets for %s tag(%s)", factory, listByTag)

	if listProd && len(listByTag) == 0 {
		subcommands.DieNotNil(errors.New("--production flag requires --by-tag flag"))
	}

	if listRaw {
		if listProd {
			meta, err := api.ProdTargetsGet(factory, listByTag, true)
			subcommands.DieNotNil(err)
			bytes, err := json.Marshal(meta)
			subcommands.DieNotNil(err)
			os.Stdout.Write(bytes)
		} else {
			body, err := api.TargetsListRaw(factory)
			subcommands.DieNotNil(err)
			os.Stdout.Write(*body)
		}
		return
	}

	var targets data.Files
	if listProd {
		meta, err := api.ProdTargetsGet(factory, listByTag, true)
		subcommands.DieNotNil(err)
		targets = meta.Signed.Targets
	} else {
		var err error
		targets, err = api.TargetsList(factory)
		subcommands.DieNotNil(err)
	}

	var keys []string
	listing := make(map[string]*targetListing)
	for _, target := range targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		if custom.TargetFormat != "OSTREE" {
			logrus.Debugf("Skipping non-ostree target: %v", target)
			continue
		}
		if len(listByTag) > 0 {
			found := false
			for _, t := range custom.Tags {
				if t == listByTag {
					found = true
					break
				}
			}
			if !found {
				logrus.Debugf("Skipping tag: %v", target)
				continue
			}
		}
		ver, err := strconv.Atoi(custom.Version)
		if err != nil {
			panic(fmt.Sprintf("Invalid version: %v. Error: %s", target, err))
		}
		key := fmt.Sprintf("%d-%s", ver, strings.Join(custom.Tags, ","))
		build, ok := listing[key]
		if ok {
			build.hardwareIds = append(build.hardwareIds, custom.HardwareIds...)
			//TODO assert list of docker-apps is the same
		} else {
			set := make(map[string]bool)
			var apps []string
			for app := range custom.ComposeApps {
				if _, ok := set[app]; !ok {
					apps = append(apps, app)
				}
			}
			sort.Strings(apps)
			keys = append(keys, key)
			origin := ""
			if len(custom.OrigUri) > 0 {
				parts := strings.Split(custom.OrigUri, "/")
				origin = parts[len(parts)-1]
			}
			listing[key] = &targetListing{
				version:      ver,
				hardwareIds:  custom.HardwareIds,
				tags:         custom.Tags,
				apps:         apps,
				origin:       origin,
				manifestSha:  custom.LmpManifestSha,
				overridesSha: custom.OverridesSha,
				containerSha: custom.ContainersSha,
			}
		}
	}

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

	sort.Sort(byTargetKey(keys))
	for _, key := range keys {
		l := listing[key]
		for idx, col := range showColumns {
			col := Columns[col]
			row[idx] = col.Formatter(l)
		}
		t.AddLine(row...)
	}
	t.Print()
}
