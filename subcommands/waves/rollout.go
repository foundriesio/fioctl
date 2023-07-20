package waves

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rollout := &cobra.Command{
		Use:   "rollout <wave>",
		Short: "Roll out a wave to a subset of production devices",
		Long: `Roll out a wave to a subset of production devices matching a wave's tag.
Upon rollout a wave becomes available as an update source for a given subset of production devices.
A rollout is not instant, rather each device will update to the wave's targets at some point.
The exact update time is determined by many factors:
device up and down lifecycle, its update schedule, networking between a device and update servers, etc.
At least one command flag is required to limit the subset of devices to roll out to.
If you want to roll out to all matching devices in a factory, please, use the "complete" command.`,
		Run:  doRolloutWave,
		Args: cobra.RangeArgs(1, 2),
		Example: `
Rollout a wave to all devices in the "us-east" device group:
$ fioctl waves rollout --group us-east

Rollout a wave to 10 devices in the "us-east" device group:
$ fioctl waves rollout --group us-east --limit=10

Rollout a wave to 2 specific devices in the "us-east" device group:
$ fioctl waves rollout --group us-east --uuids=uuid1,uuid2

Rollout a wave to 10% devices in your factory:
$ fioctl waves rollout --limit=10%

Rollout a wave to specific devices in your factory, device UUIDs provided by a file:
$ fioctl waves rollout --uuids=@/path/to/file

Rollout a wave to 10% of specific devices in your factory, device UUIDs provided by a file:
$ fioctl waves rollout --uuids=@/path/to/file --limit=10%

In all of the above examples:
- When using the "uuids" flag, each device in a list is verified to match wave requirements.
  In addition, if the "group" flag is provided, each device must also belong to that device group.
- When using the "limit" flag, a list of rolled out devices is auto-selected by the API.
  The most recently active devices have a higher chance to get into this selection.
  A device is excluded from the selection, if a wave was already rolled out to it earlier.
- Using both "uuids" and "limit" flags constrains auto-selection to a given device list.
  This can be combined with the "group" flag to further constrain it to a given device group.
- The following characters are supported as a separator for the device list in the "uuids" flag:
  a comma (","), a semicolon (";"), a pipe ("|"), white space, tabs, and line breaks.
  The user is responsible for properly escaping these characters in a shell script.
  It is recommended to pass a list of UUIDs via a file if their number is big enough.
`,
	}
	rollout.Flags().StringP("group", "g", "", "A device group to roll out a wave to")
	rollout.Flags().StringP("limit", "l", "",
		`A number of devices to roll out a wave to.
It can be an exact number (e.g. 10), or as a percentage of all matching devices (e.g. 10%).
An actual number of rolled out devices can be less than the specified value.
A maximum number of devices rolled out using this flag cannot exceed 10000.`,
	)
	rollout.Flags().StringP("uuids", "", "",
		`A comma-separated list of exact device UUIDs to roll out a wave to.
Also accepts a filename containing a comma-separated list via "--uuids=@path/to/file.name".
A maximum number of devices rolled out using this flag cannot exceed 10000.`,
	)
	rollout.Flags().BoolP("dry-run", "", false,
		"Only show what would happen without an actual rollout. Most useful with --print-xxx flags.")
	rollout.Flags().BoolP("print-uuids", "", false,
		"Print UUIDs of devices to which a wave was rolled out (would be rolled out with --dry-run).")
	rollout.Flags().BoolP("print-names", "", false,
		"Print names of devices to which a wave was rolled out (would be rolled out with --dry-run).")
	cmd.AddCommand(rollout)
}

func doRolloutWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	wave := args[0]
	group := readGroup(cmd, args)
	limit, percentage := readLimit(cmd)
	uuids := readUuids(cmd)
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	printUuids, _ := cmd.Flags().GetBool("print-uuids")
	printNames, _ := cmd.Flags().GetBool("print-names")

	isFullRollout := len(uuids) == 0 && limit == 0 && percentage == 0

	if len(group) == 0 && isFullRollout {
		subcommands.DieNotNil(errors.New(
			"One of the following flags must be set: group, limit, percentage, uuids\n" + cmd.UsageString(),
		))
	}

	selector := getDebugSelector(group, uuids, limit, percentage)
	logrus.Debugf("Rolling out a wave %s for %s to %s", wave, factory, selector)

	options := client.WaveRolloutOptions{
		Group:      group,
		Limit:      limit,
		Percentage: percentage,
		Uuids:      uuids,
		DryRun:     dryRun,
		PrintUuids: printUuids,
		PrintNames: printNames,
	}
	res, err := api.FactoryRolloutWave(factory, wave, options)
	subcommands.DieNotNil(err)

	if len(group) == 0 {
		if dryRun {
			fmt.Printf("A wave would be rolled out to %d devices in your factory.\n", res.DeviceNum)
		} else {
			fmt.Printf("A wave is being rolled out to %d devices in your factory.\n", res.DeviceNum)
		}
	} else {
		if dryRun {
			fmt.Printf("A wave would be rolled out to %d devices in group \"%s\".\n", res.DeviceNum, group)
			if isFullRollout {
				fmt.Printf("It would also be rolled out to all new devices in group \"%s\".\n", group)
			}
		} else {
			fmt.Printf("A wave is being rolled out to %d devices in group \"%s\".\n", res.DeviceNum, group)
			if isFullRollout {
				fmt.Printf("It will also be rolled out to all new devices in group \"%s\".\n", group)
			}
		}
	}
	if (printUuids && len(res.DeviceUuids) == 0) || (printNames && len(res.DeviceNames) == 0) {
		// This can happen when rolling out to a large group without any limit specified.
		fmt.Println("A full list of these devices exceeds 10000, so it cannot be printed.")
	} else if printUuids || printNames {
		fmt.Println("Below is a full list of these devices:")
		for i := 0; i < res.DeviceNum; i++ {
			if i > 0 {
				fmt.Print(", ")
			}
			if printUuids {
				fmt.Print(res.DeviceUuids[i])
				if printNames {
					fmt.Printf(" (%s)", res.DeviceNames[i])
				}
			} else if printNames {
				fmt.Print(res.DeviceNames[i])
			}
		}
		fmt.Println("")
	}
}

func readGroup(cmd *cobra.Command, args []string) string {
	// Backward-compatible reader: new way - named flag, old way - positional flag
	group, _ := cmd.Flags().GetString("group")
	if len(args) > 1 {
		if len(group) > 0 {
			subcommands.DieNotNil(errors.New(
				"Flag \"group\" cannot be both positional and named\n" + cmd.UsageString(),
			))
		}
		group = args[1]
	}
	return group
}

func readUuids(cmd *cobra.Command) []string {
	uuids, _ := cmd.Flags().GetString("uuids")
	if len(uuids) == 0 {
		return nil
	} else if uuids[0] == '@' {
		if content, err := os.ReadFile(uuids[1:]); err != nil {
			subcommands.DieNotNil(err, "Failed to read the devices UUIDs from a file:")
		} else if len(content) == 0 {
			subcommands.DieNotNil(errors.New("Devices UUIDs file cannot be empty"))
		} else {
			uuids = string(content)
		}
	}

	// split by common list separators and all known line breaks, tabs and other white space.
	res := strings.FieldsFunc(uuids, func(c rune) bool {
		return c == ',' || c == ';' || c == '|' || unicode.IsSpace(c)
	})
	// The above splitting could produce empty values - filter them out (inplace)
	var i int = 0
	for _, r := range res {
		if len(r) > 0 {
			if len(r) > 60 {
				if len(r) > 100 {
					r = r[:100] + "... (cropped)"
				}
				subcommands.DieNotNil(fmt.Errorf("Device uuid value is too long, limit is 60: %s", r))
			}
			res[i] = r
			i += 1
		}
	}
	res = res[:i]

	if len(res) > 10000 {
		// API will not accept more UUIDs anyway, so why should we try
		subcommands.DieNotNil(fmt.Errorf(
			"Device uuids list contains %d items, limit is 10000", len(res),
		))
	}
	return res
}

func readLimit(cmd *cobra.Command) (limit int, percentage int) {
	value, _ := cmd.Flags().GetString("limit")
	if len(value) > 0 {
		var err error
		if value[len(value)-1] == '%' {
			if percentage, err = strconv.Atoi(value[0 : len(value)-1]); err != nil {
				subcommands.DieNotNil(fmt.Errorf(
					`A "limit" must be either a number or a percentage: %s`, value,
				))
			}
		} else {
			if limit, err = strconv.Atoi(value); err != nil {
				subcommands.DieNotNil(fmt.Errorf(
					`A "limit" must be either a number or a percentage: %s`, value,
				))
			}
		}
	}
	return
}

func getDebugSelector(group string, uuids []string, limit, percentage int) string {
	selector := "all devices"
	if len(uuids) > 0 {
		var num = len(uuids)
		if limit > 0 && limit < num {
			num = limit
		} else if percentage > 0 && percentage < 100 {
			num = num * percentage / 100
		}
		selector = strconv.Itoa(num) + " devices"
	} else if limit > 0 {
		selector = strconv.Itoa(limit) + " devices"
	} else if percentage > 0 {
		selector = strconv.Itoa(percentage) + "% devices"
	}
	if len(group) > 0 {
		selector += " in " + group
	}
	return selector
}
