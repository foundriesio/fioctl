package waves

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "complete <wave>",
		Short: "Complete a given wave by name to make it generally available",
		Long: `Complete a given wave by name.
Once complete a wave becomes generally available as an update source for all production devices.
A subsequent wave might become a new source for a part of production devices again.`,
		Run:  doCompleteWave,
		Args: cobra.ExactArgs(1),
	})
}

func doCompleteWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	logrus.Debugf("Completing a wave %s for %s", name, factory)

	subcommands.DieNotNil(api.FactoryCompleteWave(factory, name))
}
