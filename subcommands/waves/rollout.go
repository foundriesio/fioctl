package waves

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rollout := &cobra.Command{
		Use:   "rollout <wave> -g <group>",
		Short: "Rollout a given wave to devices in a given device group",
		Long: `Rollout a given wave to devices in a given device group.
Upon rollout a wave becomes available as an update source for production devices in a specific
device group.  An rollout is not instant, but rather each device in a given group will update to
this wave's targets at some point in time which is determined by many factors: most important being
network conditions between a device and update servers, as well as a device update schedule.`,
		Run:  doRolloutWave,
		Args: cobra.RangeArgs(1, 2),
	}
	rollout.Flags().StringP("group", "g", "", "A device group to roll out a wave to")
	cmd.AddCommand(rollout)
}

func doRolloutWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	wave := args[0]
	group := readGroup(cmd, args)
	if len(group) == 0 {
		subcommands.DieNotNil(errors.New("Required flag \"group\" not set\n" + cmd.UsageString()))
	}
	logrus.Debugf("Rolling out a wave %s for %s to %s", wave, factory, group)

	options := client.WaveRolloutOptions{Group: group}
	subcommands.DieNotNil(api.FactoryRolloutWave(factory, wave, options))
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
