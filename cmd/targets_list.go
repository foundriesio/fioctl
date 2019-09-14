package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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
	listRaw    bool
	listByTag  string
	listByHwId string
)

func init() {
	targetsCmd.AddCommand(targetListCmd)
	targetListCmd.Flags().BoolVarP(&listRaw, "raw", "r", false, "Print raw targets.json")
	targetListCmd.Flags().StringVarP(&listByHwId, "by-hwid", "", "", "Only list targets that match the given hardware-id")
	targetListCmd.Flags().StringVarP(&listByTag, "by-tag", "", "", "Only list targets that match the given tag")
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
	for name, target := range targets.Signed.Targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		if len(listByHwId) > 0 && !intersectionInSlices([]string{listByHwId}, custom.HardwareIds) {
			logrus.Debugf("Target(%r) does not match hwid)", target)
			continue
		}
		if len(listByTag) > 0 && !intersectionInSlices([]string{listByTag}, custom.Tags) {
			logrus.Debugf("Target(%r) does not include tag)", target)
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
	}
}
