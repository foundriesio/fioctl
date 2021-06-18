package targets

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	pruneNoTail   bool
	pruneByTag    bool
	pruneDryRun   bool
	pruneKeepLast int
)

func init() {
	pruneCmd := &cobra.Command{
		Use:   "prune <target> [<target>...]",
		Short: "Prune target(s)",
		Run:   doPrune,
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(pruneCmd)
	pruneCmd.Flags().BoolVarP(&pruneNoTail, "no-tail", "", false, "Don't tail output of CI Job")
	pruneCmd.Flags().BoolVarP(&pruneByTag, "by-tag", "", false, "Prune all targets by tags instead of name")
	pruneCmd.Flags().IntVarP(&pruneKeepLast, "keep-last", "", 0, "Keep the last X number of builds for a tag when pruning")
	pruneCmd.Flags().BoolVarP(&pruneDryRun, "dryrun", "", false, "Only show what would be pruned")
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

func sortedListsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func doPrune(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)

	var target_names []string
	if pruneByTag {
		sort.Strings(args)
		target_names = make([]string, 0, 10)
		for name, target := range targets {
			custom, err := api.TargetCustom(target)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
			} else {
				sort.Strings(custom.Tags)
				if sortedListsMatch(args, custom.Tags) {
					target_names = append(target_names, name)
				}
			}
		}
		if pruneKeepLast > 0 {
			versions := make(map[int][]string)
			for _, name := range target_names {
				parts := strings.SplitN(name, "lmp-", 2)
				if len(parts) != 2 {
					fmt.Printf("Unable to decode target name: %s\n", name)
					os.Exit(1)
				}
				verI, err := strconv.Atoi(parts[1])
				subcommands.DieNotNil(err)
				vals, ok := versions[verI]
				if ok {
					versions[verI] = append(vals, name)
				} else {
					versions[verI] = []string{name}
				}
			}
			var versionNums []int
			for k := range versions {
				versionNums = append(versionNums, k)
			}
			sort.Sort(sort.Reverse(sort.IntSlice(versionNums)))
			if len(versionNums) > pruneKeepLast {
				for i := 0; i < pruneKeepLast; i++ {
					delete(versions, versionNums[i])
				}
				target_names = []string{}
				for _, names := range versions {
					target_names = append(target_names, names...)
				}
			} else {
				fmt.Println("Nothing to prune")
				os.Exit(0)
			}
		}
	} else {
		for _, name := range args {
			if _, ok := targets[name]; !ok {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
		target_names = args
	}

	fmt.Printf("Deleting targets:\n %s\n", strings.Join(target_names, "\n "))
	if pruneDryRun {
		fmt.Println("Dry run, exiting")
		return
	}

	jobservUrl, webUrl, err := api.TargetDeleteTargets(factory, target_names)
	subcommands.DieNotNil(err)
	fmt.Printf("CI URL: %s\n", webUrl)
	if !pruneNoTail {
		api.JobservTail(jobservUrl)
	}
}
