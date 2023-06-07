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
		Use:   "status [<wave>]",
		Short: "Show a status for a given wave by name",
		Long: `Show a status for a given wave by name.
When no wave name is provided - show a status for a currently active wave.

For an active wave this command shows an overview of a wave,
followed by an overview of device groups participating in a wave,
and after that a detailed information for each rollout group.

For finished waves a detailed per group information is not shown as it is no more relevant.

When counting a total number of devices participating in a wave,
each production device that has a tag equal to a wave tag counts.
In particular, devices outside rollout groups also count if they satisfy this condition.

All other numbers are calculated relative to this total number.
For example, online devices in each group are counted among only those production devices,
that belong to a given group and also have a tag equal to a group tag.
This number can be lower than a total number of online devices in this group.

In a device group overview all wave rollout groups are shown first in an order of rollout time.
After that follow other groups that have devices with matching tag (if they contain at least one such device).
The last row is for devices not belonging to any group (if at least one such device matches a wave tag).

A number of updated devices depends onto a wave status:
For active wave it is a number of devices in rollout groups with target version >= wave version.
For finished waves it is a number of all devices with target version >= wave version.

Meaning of scheduled vs unscheduled (for update) device number also depends onto a wave status:
For active wave, scheduled for update are devices in rollout groups with target version < wave version.
For complete wave, scheduled are all devices (regardless a group) with target version < wave version.
For canceled wave, all devices with target version < wave version are unscheduled (scheduled is always zero).

For finished waves all numbers are calculated for a current date (not a date of a wave finishing).
This can be used to monitor how an update progresses after a wave has been complete.
`,
		Run:  doShowWaveStatus,
		Args: cobra.RangeArgs(0, 1),
	}
	cmd.AddCommand(showCmd)
	showCmd.Flags().Int("offline-threshold", 4, "Consider device 'OFFLINE' if not seen in the last X hours")
}

func doShowWaveStatus(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	offlineThreshold, _ := cmd.Flags().GetInt("offline-threshold")
	name := "$active$"
	if len(args) > 0 {
		name = args[0]
		logrus.Debugf("Showing a wave %s status for %s", name, factory)
	} else {
		logrus.Debugf("Showing an active wave status for %s", factory)
	}

	status, err := api.FactoryWaveStatus(factory, name, offlineThreshold)
	subcommands.DieNotNil(err)

	fmt.Printf("Wave '%s' for tag '%s' version %d is %s\n",
		status.Name, status.Tag, status.Version, status.Status)
	fmt.Println()
	if status.Status != "active" {
		fmt.Println("A device information is shown for a current time, not for a time when a wave was finished")
		fmt.Println()
	}
	fmt.Printf("Created At: \t%s\n", status.CreatedAt)
	if status.FinishedAt != "" {
		fmt.Printf("Finished At: \t%s\n", status.FinishedAt)
	}

	t := subcommands.Tabby(0)
	t.AddLine(fmt.Sprintf("Device Groups on Tag '%s':", status.Tag),
		len(status.RolloutGroups)+len(status.OtherGroups))
	t.AddLine("Device Groups Rollout:", len(status.RolloutGroups))
	t.AddLine(fmt.Sprintf("Devices on Tag '%s':", status.Tag), status.TotalDevices)
	t.AddLine("Devices Updated:", status.UpdatedDevices)
	t.AddLine("Devices Scheduled for Update:", status.ScheduledDevices)
	t.AddLine("Devices Not Scheduled:", status.UnscheduledDevices)
	t.Print()
	fmt.Println()

	if len(status.RolloutGroups) > 0 || len(status.OtherGroups) > 0 {
		unscheduledMessage := "At Wave Completion"
		if status.Status == "canceled" {
			unscheduledMessage = "Never (Wave Canceled)"
		} else if status.Status == "complete" {
			unscheduledMessage += " (Done)"
		}

		t = subcommands.Tabby(0, "GROUP", "TOTAL", "UPDATED", "NEED UPDATE", "SCHEDULED", "ONLINE", "ROLLOUT AT")
		for _, group := range status.RolloutGroups {
			t.AddLine(
				group.Name, group.DevicesTotal, group.DevicesOnWave+group.DevicesOnNewer,
				group.DevicesOnOlder, group.DevicesScheduled, group.DevicesOnline, group.RolloutAt)
		}
		for _, group := range status.OtherGroups {
			if group.Name == "" {
				group.Name = "(No Group)"
			}
			t.AddLine(
				group.Name, group.DevicesTotal, group.DevicesOnWave+group.DevicesOnNewer,
				group.DevicesOnOlder, group.DevicesScheduled, group.DevicesOnline, unscheduledMessage)
		}
		t.Print()
	}

	if len(status.RolloutGroups) > 0 && status.Status == "active" {
		fmt.Println("\nOrphan target versions below are marked with a star (*)")
		fmt.Println("Wave target version below is marked with an arrow (<-)")
		for _, group := range status.RolloutGroups {
			fmt.Printf("\n## Device Group: %s\n", group.Name)
			t = subcommands.Tabby(1, "TARGET", "DEVICES", "INSTALLING", "DETAILS")
			for _, tgt := range group.Targets {
				var mark, details string
				if tgt.Version == status.Version {
					mark = "<-"
				} else if tgt.IsOrphan {
					mark = "*"
				}
				if tgt.Version > 0 {
					details = fmt.Sprintf("`fioctl targets show %d`", tgt.Version)
				}
				t.AddLine(fmt.Sprintf("%-6d%2s", tgt.Version, mark), tgt.Devices, tgt.Reinstalling, details)
			}
			t.Print()
		}
	}
}
