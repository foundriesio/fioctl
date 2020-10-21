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
}

var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Manage Public Key Infrastructure for your device gateway",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api = subcommands.Login(cmd)
	},
	Long: `Every factory can have its own dedicated device gateway. This allows customers
to own the PKI infrastructure of their factory. This infrastructure is used
to manage mutual TLS between your devices and the Foundries.io device gateway.`,
}

func NewCommand() *cobra.Command {
	subcommands.RequireFactory(caCmd)
	cmd.AddCommand(caCmd)
	return cmd
}
