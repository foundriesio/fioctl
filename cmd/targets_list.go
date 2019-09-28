package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
)

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List targets.",
	Run:   doTargetsList,
}

var (
	listRaw    bool
	listByTag  string
	listByHwId string
	listByVer  bool
)

type targetCustom struct {
	target tuf.FileMeta
	custom *client.TufCustom
}

func init() {
	targetsCmd.AddCommand(targetListCmd)
	targetListCmd.Flags().BoolVarP(&listRaw, "raw", "r", false, "Print raw targets.json")
	targetListCmd.Flags().StringVarP(&listByHwId, "by-hwid", "", "", "Only list targets that match the given hardware-id")
	targetListCmd.Flags().StringVarP(&listByTag, "by-tag", "", "", "Only list targets that match the given tag")
	targetListCmd.Flags().BoolVarP(&listByVer, "by-version", "", false, "Group by \"versions\" so that each version shows the hwids and shas for it")
}

func printSorted(targets map[string][]targetCustom) {
	keys := make([]string, 0, len(targets))
	for k := range targets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Println("=", k)
		for _, tc := range targets[k] {
			hash := hex.EncodeToString(tc.target.Hashes["sha256"])
			fmt.Printf("\t%s / %s\n", strings.Join(tc.custom.HardwareIds, ","), hash)
			if len(tc.custom.Tags) > 0 {
				fmt.Printf("\t\ttags:%s\n", strings.Join(tc.custom.Tags, ","))
			}
			if len(tc.custom.DockerApps) > 0 {
				fmt.Printf("\t\tdocker-apps:")
				idx := 0
				for name, app := range tc.custom.DockerApps {
					if idx > 0 {
						fmt.Printf(",")
					}
					fmt.Printf("%s=%s", name, app.FileName)
					idx += 1
				}
				fmt.Printf("\n")
			}
		}
	}
}


func doTargetsList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing targets for %s hwid(%s) tag(%s)", factory, listByHwId, listByTag)

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

	verTargets := map[string][]targetCustom{}
	for name, target := range targets.Signed.Targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		if len(listByHwId) > 0 && !intersectionInSlices([]string{listByHwId}, custom.HardwareIds) {
			logrus.Debugf("Target(%v) does not match hwid)", target)
			continue
		}
		if len(listByTag) > 0 && !intersectionInSlices([]string{listByTag}, custom.Tags) {
			logrus.Debugf("Target(%v) does not include tag)", target)
			continue
		}
		if listByVer {
			verTargets[custom.Version] = append(verTargets[custom.Version], targetCustom{target, custom})
			continue
		}
		fmt.Println("=", name)
		hash := hex.EncodeToString(target.Hashes["sha256"])
		fmt.Printf("\tsha256:%s\n", hash)
		fmt.Printf("\tversion:%s\n", custom.Version)
		fmt.Printf("\thwids:%s\n", strings.Join(custom.HardwareIds, ","))
		fmt.Printf("\tformat:%s\n", custom.TargetFormat)
		if len(custom.Tags) > 0 {
			fmt.Printf("\ttags:%s\n", strings.Join(custom.Tags, ","))
		}
		if len(custom.DockerApps) > 0 {
			fmt.Printf("\tdocker-apps:")
			idx := 0
			for name, app := range custom.DockerApps {
				if idx > 0 {
					fmt.Printf(",")
				}
				fmt.Printf("%s=%s", name, app.FileName)
				idx += 1
			}
			fmt.Printf("\n")
		}
	}
	if listByVer {
		printSorted(verTargets)
	}
}
