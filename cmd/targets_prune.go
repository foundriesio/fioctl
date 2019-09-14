package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var targetsPruneCmd = &cobra.Command{
	Use:   "prune <target> [<target>...]",
	Short: "Prune target(s)",
	Run:   doTargetsPrune,
	Args:  cobra.MinimumNArgs(1),
}

var (
	pruneNoTail bool
	pruneByTag  bool
	pruneDryRun bool
)

func init() {
	targetsCmd.AddCommand(targetsPruneCmd)
	targetsPruneCmd.Flags().BoolVarP(&pruneNoTail, "no-tail", "", false, "Don't tail output of CI Job")
	targetsPruneCmd.Flags().BoolVarP(&pruneByTag, "by-tag", "", false, "Prune all targets by tags instead of name")
	targetsPruneCmd.Flags().BoolVarP(&pruneDryRun, "dryrun", "", false, "Only show what would be pruned")
}

func intersectionInSlices(list1, list2 []string) bool {
	for _, a := range list1 {
		for _, b := range list2 {
			if b == a {
				return true
			}
		}
	}
	return false
}

func doTargetsPrune(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	targets, err := api.TargetsList(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	var target_names []string
	if pruneByTag {
		target_names = make([]string, 0, 10)
		for name, target := range targets.Signed.Targets {
			custom, err := api.TargetCustom(target)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
			} else {
				if intersectionInSlices(args, custom.Tags) {
					target_names = append(target_names, name)
				}
			}
		}
	} else {
		for _, name := range args {
			if _, ok := targets.Signed.Targets[name]; !ok {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
		target_names = args
	}
	fmt.Printf("Deleting targets: %s\n", strings.Join(target_names, ","))
	if pruneDryRun {
		fmt.Println("Dry run, exiting")
		return
	}

	url, err := api.TargetDeleteTargets(factory, target_names)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CI URL: %s\n", url)
	if !pruneNoTail {
		api.JobservTail(url)
	}
}
