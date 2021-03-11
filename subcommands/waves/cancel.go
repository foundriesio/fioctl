package waves

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "cancel <wave>",
		Short: "Cancel a given wave by name",
		Long: `Cancel a given wave by name.
Once canceled a wave is no longer available as an update source for production devices.
However, those devices that has already updated to a wave version
will remain on that version until a new version is rolled out.`,
		Run:  doCancelWave,
		Args: cobra.ExactArgs(1),
	})
}

func doCancelWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	logrus.Debugf("Canceling a wave %s for %s", name, factory)

	subcommands.DieNotNil(api.FactoryCancelWave(factory, name))
}
