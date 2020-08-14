package targets

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"
)

var (
	pruneNoTail bool
	pruneByTag  bool
	pruneDryRun bool
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

func findUnusedApps(targets *tuf.SignedTargets, deleted_list []string) []string {
	apps := make(map[string]int)
	referenced := make([]string, 0, 100)
	for name, target := range targets.Signed.Targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
		} else {
			if custom.TargetFormat == "BINARY" && strings.HasSuffix(custom.Name, ".dockerapp") {
				apps[name] = 0
			}
			if !intersectionInSlices([]string{name}, deleted_list) {
				for _, app := range custom.DockerApps {
					if len(app.FileName) > 0 {
						referenced = append(referenced, app.FileName)
					}
				}
			}
		}
	}

	for _, app := range referenced {
		delete(apps, app)
	}

	unused := make([]string, len(apps))
	i := 0
	for app := range apps {
		unused[i] = app
		i++
	}
	return unused
}

func doPrune(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	targets, err := api.TargetsList(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	var target_names []string
	if pruneByTag {
		sort.Strings(args)
		target_names = make([]string, 0, 10)
		for name, target := range targets.Signed.Targets {
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
	} else {
		for _, name := range args {
			if _, ok := targets.Signed.Targets[name]; !ok {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
		target_names = args
	}

	unused := findUnusedApps(targets, target_names)
	target_names = append(target_names, unused...)

	fmt.Printf("Deleting targets:\n %s\n", strings.Join(target_names, "\n "))
	if pruneDryRun {
		fmt.Println("Dry run, exiting")
		return
	}

	jobservUrl, webUrl, err := api.TargetDeleteTargets(factory, target_names)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CI URL: %s\n", webUrl)
	if !pruneNoTail {
		api.JobservTail(jobservUrl)
	}
}
