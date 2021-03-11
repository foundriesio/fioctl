package targets

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

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	listRaw   bool
	listByTag string
)

// Represents the details we use for displaying a single OTA "build"
type targetListing struct {
	version     int
	hardwareIds []string
	tags        []string
	apps        []string
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

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List targets.",
		Run:   doList,
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listRaw, "raw", "r", false, "Print raw targets.json")
	listCmd.Flags().StringVarP(&listByTag, "by-tag", "", "", "Only list targets that match the given tag")
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing targets for %s tag(%s)", factory, listByTag)

	if listRaw {
		body, err := api.TargetsListRaw(factory)
		subcommands.DieNotNil(err)
		os.Stdout.Write(*body)
		return
	}
	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)

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
			for app := range custom.DockerApps {
				apps = append(apps, app)
				set[app] = true
			}
			for app := range custom.ComposeApps {
				if _, ok := set[app]; !ok {
					apps = append(apps, app)
				}
			}
			keys = append(keys, key)
			listing[key] = &targetListing{
				version:     ver,
				hardwareIds: custom.HardwareIds,
				tags:        custom.Tags,
				apps:        apps,
			}
		}
	}

	t := tabby.New()
	t.AddHeader("VERSION", "TAGS", "APPS", "HARDWARE IDs")

	sort.Sort(byTargetKey(keys))
	for _, key := range keys {
		l := listing[key]
		tags := strings.Join(l.tags, ",")
		apps := strings.Join(l.apps, ",")
		hwids := strings.Join(l.hardwareIds, ",")
		t.AddLine(l.version, tags, apps, hwids)
	}
	t.Print()
}
