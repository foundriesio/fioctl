package waves

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	showCmd := &cobra.Command{
		Use:   "show <wave>",
		Short: "Show a given wave by name",
		Run:   doShowWave,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(showCmd)
	showCmd.Flags().BoolP("show-targets", "s", false, "Show wave targets")
}

func doShowWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	showTargets, _ := cmd.Flags().GetBool("show-targets")
	logrus.Debugf("Showing a wave %s for %s", name, factory)

	wave, err := api.FactoryGetWave(factory, name, showTargets)
	subcommands.DieNotNil(err)

	fmt.Printf("Name: \t\t%s\n", wave.Name)
	fmt.Printf("Version: \t%s\n", wave.Version)
	fmt.Printf("Tag: \t\t%s\n", wave.Tag)
	fmt.Printf("Status: \t%s\n", wave.Status)

	fmt.Printf("Created At: \t%s\n", wave.ChangeMeta.CreatedAt)
	if len(wave.ChangeMeta.CreatedBy) > 0 {
		fmt.Printf("Created By: \t%s\n", wave.ChangeMeta.CreatedBy)
	}
	if len(wave.RolloutGroups) > 0 {
		groupRefs := sortRolloutGroups(wave.RolloutGroups)
		firstLine := true
		for _, ref := range groupRefs {
			formatLine := "\t\t%s: rollout to device group %s"
			if firstLine {
				firstLine = false
				formatLine = "Rollout At: \t%s: rollout to device group %s"
			}
			groupName := ref.GroupName
			if groupName == "" {
				// A group has been deleted, only a reference still exists - we cannot track down a name
				groupName = "<deleted group>"
			}
			line := fmt.Sprintf(formatLine, ref.CreatedAt, groupName)
			if len(ref.CreatedBy) > 0 {
				line += " by " + ref.CreatedBy
			}
			fmt.Println(line)
		}
	}
	if wave.ChangeMeta.UpdatedAt != "" {
		fmt.Printf("Finished At: \t%s\n", wave.ChangeMeta.UpdatedAt)
	}
	if wave.ChangeMeta.UpdatedBy != "" {
		fmt.Printf("Finished By: \t%s\n", wave.ChangeMeta.UpdatedBy)
	}

	if showTargets {
		fmt.Printf("Targets:\n")
		data, _ := json.MarshalIndent(wave.Targets, "  ", "  ")
		fmt.Println("  " + string(data))
	}
}

func sortRolloutGroups(groupMap map[string]client.WaveRolloutGroupRef) []client.WaveRolloutGroupRef {
	groupRefs := make([]client.WaveRolloutGroupRef, 0, len(groupMap))
	for _, ref := range groupMap {
		groupRefs = append(groupRefs, ref)
	}
	sort.Slice(groupRefs, func(i, j int) bool {
		// Time is in RFC3339 format i.e. with zero padding, so it compares properly
		return groupRefs[i].CreatedAt < groupRefs[j].CreatedAt
	})
	return groupRefs
}
