package keys

import (
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	api *client.Api
)

var cmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage keys in use by your factory fleet",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
}

var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Manage Public Key Infrastructure for your device gateway",
	Long: `Every factory can have its own dedicated device gateway. This allows customers
to own the PKI infrastructure of their factory. This infrastructure is used
to manage mutual TLS between your devices and the Foundries.io device gateway.`,
}

var tufCmd = &cobra.Command{
	Use:   "tuf",
	Short: "Manage The Update Framework Keys for your factory",
	Long: `These sub-commands allow you to manage your Factory's TUF private keys
to ensure that you are in complete control of your OTA metadata.`,
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)
	cmd.AddCommand(caCmd)
	cmd.AddCommand(tufCmd)
	return cmd
}
