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

func init() {
	targetsCmd.AddCommand(targetListCmd)
	targetListCmd.PersistentFlags().BoolP("raw", "r", false, "Print raw targets.json")
	if err := viper.BindPFlags(targetListCmd.PersistentFlags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func doTargetsList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing targets for %s", factory)

	if viper.GetBool("raw") {
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
	for name := range targets.Signed.Targets {
		fmt.Println("=", name)
		target := targets.Signed.Targets[name]
		hash := hex.EncodeToString(target.Hashes["sha256"])
		fmt.Printf("\tsha256:%s\n", hash)
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
		} else {
			fmt.Printf("\tversion:%s\n", custom.Version)
			fmt.Printf("\tformat:%s\n", custom.TargetFormat)
			if len(custom.Tags) > 0 {
				fmt.Printf("\ttags:%s\n", strings.Join(custom.Tags, ","))
			}
		}
	}
}
