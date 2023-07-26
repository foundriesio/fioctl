package waves

import (
	"github.com/docker/go/canonical/json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
	tuf "github.com/theupdateframework/notary/tuf/data"
)

func init() {
	signCmd := &cobra.Command{
		Use:   "sign <wave>",
		Short: "Sign an existing wave targets with additional key",
		Long: `Sign an existing wave targets with additional key.

This command is only needed when your TUF root requires more than 1 signature for production targets.
In this case, you cannot roll out or complete a wave before it has enough signatures.`,
		Run:  doSignWave,
		Args: cobra.ExactArgs(1),
	}
	cmd.AddCommand(signCmd)
	signCmd.Flags().StringP("keys", "k", "", "Path to <offline-creds.tgz> used to sign wave targets.")
	_ = signCmd.MarkFlagRequired("keys")
}

func doSignWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	offlineKeys := readOfflineKeys(cmd)

	wave, err := api.FactoryGetWave(factory, name, true)
	subcommands.DieNotNil(err)

	var targets tuf.Signed
	subcommands.DieNotNil(json.Unmarshal(*wave.Targets, &targets))
	meta, err := json.MarshalCanonical(targets.Signed)
	subcommands.DieNotNil(err)

	signatures := signTargets(meta, factory, offlineKeys)
	subcommands.DieNotNil(api.FactorySignWave(factory, name, signatures))
}
