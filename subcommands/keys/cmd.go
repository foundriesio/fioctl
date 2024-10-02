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
	Short: "Manage keys in use by your Factory fleet",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
}

var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Manage Public Key Infrastructure for your device gateway",
	Long: `Every Factory can have its own dedicated device gateway. This allows you
to own the PKI infrastructure of your Factory. This infrastructure is used
to manage mutual TLS between your devices and the Foundries.io device gateway.`,
}

var estCmd = &cobra.Command{
	Use:   "est",
	Short: "Manage the Foundries EST server TLS keypair for your Factory",
	Long: `This command allows users to authorize Foundries.io to run an EST 7030 server
for device certificate renewal.`,
	Hidden: true,
}

var tufCmd = &cobra.Command{
	Use:   "tuf",
	Short: "Manage The Update Framework Keys for your Factory",
	Long: `These sub-commands allow you to manage your Factory's TUF private keys
to ensure that you are in complete control of your OTA metadata.`,
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(cmd)
	cmd.AddCommand(caCmd)
	cmd.AddCommand(estCmd)
	cmd.AddCommand(tufCmd)
	return cmd
}
