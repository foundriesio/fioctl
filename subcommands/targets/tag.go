package targets

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	tagTags      string
	tagAppend    bool
	tagNoTail    bool
	tagByVersion bool
)

func init() {
	var tagCmd = &cobra.Command{
		Use:   "tag <target> [<target>...]",
		Short: "Apply a comma separated list of tags to one or more targets.",
		Example: `
  # Promote Target #42 currently tagged as master
  fioctl targets tag --tags master,promoted --by-version 42

  # Tag a specific Target by name
  fioctl targets tag --tags master,testing intel-corei7-64-lmp-42`,
		Run:  doTag,
		Args: cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(tagCmd)
	tagCmd.Flags().StringVarP(&tagTags, "tags", "T", "", "comma,separate,list")
	tagCmd.Flags().BoolVarP(&tagAppend, "append", "", false, "Append the given tags rather than set them")
	tagCmd.Flags().BoolVarP(&tagNoTail, "no-tail", "", false, "Don't tail output of CI Job")
	tagCmd.Flags().BoolVarP(&tagByVersion, "by-version", "", false, "Apply tags to all targets matching the given version(s)")
	tagCmd.Flags().BoolVarP(&dryRun, "dryrun", "", false, "Just show the changes that would be applied")
}

func Set(a, b []string) []string {
	unique := make([]string, 0, len(a)+len(b))
	items := make(map[string]bool, len(a))
	for i, lst := 0, a; i < 2; i, lst = i+1, b {
		for _, item := range lst {
			if ok := items[item]; !ok {
				unique = append(unique, item)
				items[item] = true
			}
		}
	}
	return unique
}

func doTag(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	tags := strings.Split(tagTags, ",")

	// Make sure all tags are unique, no accidental repeats/typos
	if len(tags) != len(Set(tags, nil)) {
		subcommands.DieNotNil(fmt.Errorf("Invalid list of tags. An item was repeated in %s", tags))
	}

	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)

	updates := make(client.UpdateTargets)

	if tagByVersion {
		for name, target := range targets {
			custom, err := api.TargetCustom(target)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
			} else {
				if intersectionInSlices([]string{custom.Version}, args) {
					targetTags := tags
					if tagAppend {
						targetTags = Set(custom.Tags, tags)
					}
					updates[name] = client.UpdateTarget{
						Custom: client.TufCustom{Tags: targetTags},
					}
					fmt.Printf("Changing tags of %s from %s -> %s\n", name, custom.Tags, targetTags)
				}
			}
		}
		if len(updates) == 0 {
			fmt.Println("ERROR: no targets found matching the given versions")
			os.Exit(1)
		}
	} else {
		for _, name := range args {
			if target, ok := targets[name]; ok {
				custom, err := api.TargetCustom(target)
				subcommands.DieNotNil(err)
				targetTags := tags
				if tagAppend {
					targetTags = Set(custom.Tags, tags)
				}
				updates[name] = client.UpdateTarget{
					Custom: client.TufCustom{Tags: targetTags},
				}
				fmt.Printf("Changing tags of %s from %s -> %s\n", name, custom.Tags, targetTags)
			} else {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
	}

	if dryRun {
		data, err := subcommands.MarshalIndent(updates, "  ", "  ")
		subcommands.DieNotNil(err)
		fmt.Println(string(data))
		return
	}

	jobServUrl, webUrl, err := api.TargetUpdateTags(factory, updates)
	subcommands.DieNotNil(err)
	fmt.Printf("CI URL: %s\n", webUrl)
	if !tagNoTail {
		api.JobservTail(jobServUrl)
	}
}
