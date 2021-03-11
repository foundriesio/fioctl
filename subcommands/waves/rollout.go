package waves

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "rollout <wave> <group>",
		Short: "Rollout a given wave to devices in a given device group",
		Long: `Rollout a given wave to devices in a given device group.
Upon rollout a wave becomes available as an update source for production devices in a specific
device group.  An rollout is not instant, but rather each device in a given group will update to
this wave's targets at some point in time which is determined by many factors: most important being
network conditions between a device and update servers, as well as a device update schedule.`,
		Run:  doRolloutWave,
		Args: cobra.ExactArgs(2),
	})
}

func doRolloutWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	wave := args[0]
	group := args[1]
	logrus.Debugf("Rolling out a wave %s for %s to %s", wave, factory, group)

	options := client.WaveRolloutOptions{Group: group}
	subcommands.DieNotNil(api.FactoryRolloutWave(factory, wave, options))
}
