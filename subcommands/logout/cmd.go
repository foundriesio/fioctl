package logout

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove Foundries.io client credentials from system",
		Run:   doLogout,
	}
}

func doLogout(cmd *cobra.Command, args []string) {
	logrus.Debug("Executing logout command")

	creds := client.NewClientCredentials(subcommands.Config.ClientCredentials)
	creds.Config.ClientId = ""
	creds.Config.ClientSecret = ""
	creds.Config.RefreshToken = ""
	creds.Config.AccessToken = ""
	subcommands.SaveOauthConfig(creds.Config)
}
