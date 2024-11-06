package waves

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	showCmd.Flags().BoolP("show-targets", "s", false, "Show Wave Targets")
}

func doShowWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	showTargets, _ := cmd.Flags().GetBool("show-targets")
	logrus.Debugf("Showing Wave %s for %s", name, factory)

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
	if len(wave.History) > 0 {
		fmt.Println("Rollout history:")
		for _, rollout := range wave.History {
			line := fmt.Sprintf("\t%s: rollout to ", rollout.RolloutAt)

			groupName := rollout.GroupName
			if !rollout.IsFactoryWide && groupName == "" {
				// A group has been deleted, only a reference still exists - we cannot track down a name
				groupName = "<deleted group>"
			}

			if rollout.IsFactoryWide {
				line += fmt.Sprintf("%d devices in Factory", rollout.DeviceNumber)
			} else if rollout.IsFullGroup {
				line += fmt.Sprintf("all devices in group %s", groupName)
			} else {
				line += fmt.Sprintf("%d devices in group %s", rollout.DeviceNumber, groupName)
			}

			if len(rollout.RolloutBy) > 0 {
				line += " by " + rollout.RolloutBy
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
		data, _ := subcommands.MarshalIndent(wave.Targets, "  ", "  ")
		fmt.Println("  " + string(data))
	}
}
