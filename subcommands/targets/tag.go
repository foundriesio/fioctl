package targets

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	tagTags      string
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
	tagCmd.Flags().BoolVarP(&tagNoTail, "no-tail", "", false, "Don't tail output of CI Job")
	tagCmd.Flags().BoolVarP(&tagByVersion, "by-version", "", false, "Apply tags to all targets matching the given version(s)")
}

func doTag(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	tags := strings.Split(tagTags, ",")
	fmt.Println(tags)

	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)

	var target_names []string
	if tagByVersion {
		target_names = make([]string, 0, 10)
		for name, target := range targets {
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
			if target, ok := targets[name]; ok {
				custom, err := api.TargetCustom(target)
				subcommands.DieNotNil(err)
				fmt.Printf("Changing tags of %s from %s -> %s\n", name, custom.Tags, tags)
			} else {
				fmt.Printf("Target(%s) not found in targets.json\n", name)
				os.Exit(1)
			}
		}
		target_names = args
	}

	jobServUrl, webUrl, err := api.TargetUpdateTags(factory, target_names, tags)
	subcommands.DieNotNil(err)
	fmt.Printf("CI URL: %s\n", webUrl)
	if !tagNoTail {
		api.JobservTail(jobServUrl)
	}
}
