package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var targetsTagCmd = &cobra.Command{
	Use:   "tag <target> [<target>...]",
	Short: "Apply a comma separated list of tags to one or more targets.",
	Run:   doTargetsTag,
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	targetsCmd.AddCommand(targetsTagCmd)
	targetsTagCmd.PersistentFlags().StringP("tags", "T", "", "comma,separate,list")
	targetsTagCmd.PersistentFlags().BoolP("no-tail", "", false, "Don't tail output of CI Job")
	targetsTagCmd.PersistentFlags().BoolP("by-version", "", false, "Apply tags to all targets matching the given version(s)")

	targetsCmd.MarkPersistentFlagRequired("tags")

	if err := viper.BindPFlags(targetsTagCmd.PersistentFlags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func doTargetsTag(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	tags := strings.Split(viper.GetString("tags"), ",")

	targets, err := api.TargetsList(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	var target_names []string
	if viper.GetBool("by-version") {
		target_names = make([]string, 0, 10)
		for name, target := range targets.Signed.Targets {
			custom, err := api.TargetCustom(target)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
			} else {
				if intersectionInSlices([]string{custom.Version}, args) {
					target_names = append(target_names, name)
					fmt.Printf("Changing tags of %s from %s -> %s\n", name, custom.Tags, tags)
				}
			}
		}
		if len(target_names) == 0 {
			fmt.Println("ERROR: no targets found matching the given versions")
			os.Exit(1)
		}
	} else {
		for _, name := range args {
			if target, ok := targets.Signed.Targets[name]; ok {
				custom, err := api.TargetCustom(target)
				if err != nil {
					fmt.Printf("ERROR: %s\n", err)
					os.Exit(1)
				}
				fmt.Printf("Changing tags of %s from %s -> %s\n", name, custom.Tags, tags)
			} else {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
		target_names = args
	}

	url, err := api.TargetUpdateTags(factory, target_names, tags)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CI URL: %s\n", url)
	if !viper.GetBool("no-tail") {
		api.JobservTail(url)
	}
}
