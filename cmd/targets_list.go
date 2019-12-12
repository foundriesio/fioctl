package cmd

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
)

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List targets.",
	Run:   doTargetsList,
}

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

func init() {
	targetsCmd.AddCommand(targetListCmd)
	targetListCmd.Flags().BoolVarP(&listRaw, "raw", "r", false, "Print raw targets.json")
	targetListCmd.Flags().StringVarP(&listByTag, "by-tag", "", "", "Only list targets that match the given tag")
}

func doTargetsList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing targets for %s tag(%s)", factory, listByTag)

	if listRaw {
		body, err := api.TargetsListRaw(factory)
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		os.Stdout.Write(*body)
		return
	}
	targets, err := api.TargetsList(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	var keys []int
	listing := make(map[int]*targetListing)
	for _, target := range targets.Signed.Targets {
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
		build, ok := listing[ver]
		if ok {
			build.hardwareIds = append(build.hardwareIds, custom.HardwareIds...)
			//TODO assert list of docker-apps is the same
			//TODO assert list of tags is the same
		} else {
			var apps []string
			for app := range custom.DockerApps {
				apps = append(apps, app)
			}
			keys = append(keys, ver)
			listing[ver] = &targetListing{
				version:     ver,
				hardwareIds: custom.HardwareIds,
				tags:        custom.Tags,
				apps:        apps,
			}
		}
	}

	t := tabby.New()
	t.AddHeader("VERSION", "TAGS", "APPS", "HARDWARE IDs")

	sort.Ints(keys)
	for _, key := range keys {
		l := listing[key]
		tags := strings.Join(l.tags, ",")
		apps := strings.Join(l.apps, ",")
		hwids := strings.Join(l.hardwareIds, ",")
		t.AddLine(l.version, tags, apps, hwids)
	}
	t.Print()
}
