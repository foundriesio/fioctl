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
		Short: "Cancel a given Wave by name",
		Long: `Cancel a given Wave by name.
Once canceled, a Wave is no longer available as an update source for production devices.
However, those already updated will remain on that version until a new version is rolled out.`,
		Run:  doCancelWave,
		Args: cobra.ExactArgs(1),
	})
}

func doCancelWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	logrus.Debugf("Canceling a Wave %s for %s", name, factory)

	subcommands.DieNotNil(api.FactoryCancelWave(factory, name))
}
