package waves

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	showCmd := &cobra.Command{
		Use:   "status [<wave>]",
		Short: "Show status for a given Wave by name",
		Long: `Show status for a given Wave by name.
When no Wave name is provided — show a status for a currently active Wave.

For an active Wave, this command shows an overview if it,
followed by an overview of device groups participating in the Wave,
and lastly, detailed information for each rollout group.

For finished Waves, detailed per group information is not shown as it is no longer relevant.

When counting the total number of devices participating in a Wave,
each production device with a tag equal to a Wave tag counts.
In particular, devices outside rollout groups also count if they satisfy this condition.

All other numbers are calculated relative to this total number.
For example, online devices in each group are counted only among those production devices
that belong to a given group and also have a tag equal to a group tag.
This number can be lower than the total number of online devices in this group.

In a device group overview, all Wave rollout groups are shown first in the order of rollout time.
Next is other groups that have devices with a matching tag — if they contain at least one such device.
The last row is for devices not belonging to any group — if at least one such device matches a Wave tag.

A number of updated devices depends onto a Wave status:
For active Waves, it is a number of devices in rollout groups with Target version >= Wave version.
For finished Waves, it is a number of all devices with Target version >= Wave version.

Meaning of scheduled vs unscheduled (for updates) device number also depends on Wave status:
For an active Wave, "scheduled for update" are devices in rollout groups with Target version < Wave version.
For a completed Wave, "scheduled" are all devices (regardless of group) with Target version < Wave version.
For a canceled Wave, all devices with Target version < Wave version are unscheduled (scheduled is always zero).

For finished Waves, all numbers are calculated for a current date (not a date of a Wave finishing).
This can be used to monitor how an update progresses after a Wave completes.
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
		logrus.Debugf("Showing Wave %s status for %s", name, factory)
	} else {
		logrus.Debugf("Showing active Wave status for %s", factory)
	}

	status, err := api.FactoryWaveStatus(factory, name, offlineThreshold)
	subcommands.DieNotNil(err)

	fmt.Printf("Wave '%s' for tag '%s' version %d is %s\n",
		status.Name, status.Tag, status.Version, status.Status)
	fmt.Println()
	if status.Status != "active" {
		fmt.Println("Device information is shown for the current time, not for when a Wave finished")
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

	hasTargets := len(status.RolloutGroups) > 0
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
			if len(group.Targets) > 0 {
				hasTargets = true
			}
		}
		t.Print()
	}

	if hasTargets && status.Status == "active" {
		fmt.Println("\nOrphan Target versions below are marked with a star (*)")
		fmt.Println("Wave Target version below is marked with an arrow (<-)")
		for _, group := range status.RolloutGroups {
			showGroupStatusTargets(group, status)
		}
		for _, group := range status.OtherGroups {
			if len(group.Targets) > 0 {
				showGroupStatusTargets(group, status)
			}
		}
	}
}

func showGroupStatusTargets(group client.RolloutGroupStatus, status *client.WaveStatus) {
	fmt.Printf("\n## Device Group: %s\n", group.Name)
	t := subcommands.Tabby(1, "TARGET", "DEVICES", "DETAILS")
	for _, tgt := range group.Targets {
		var mark, details string
		if tgt.Version == status.Version {
			mark = "<-"
		} else if tgt.IsOrphan {
			mark = "*"
		}
		if tgt.Version > 0 {
			details = fmt.Sprintf("`fioctl Targets show %d`", tgt.Version)
		}
		t.AddLine(fmt.Sprintf("%-6d%2s", tgt.Version, mark), tgt.Devices, details)
	}
	t.Print()
}
