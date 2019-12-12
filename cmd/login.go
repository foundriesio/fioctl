package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/internal/oauth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Access Foundries.io services with your client credentials",
	Run:   doLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func doLogin(cmd *cobra.Command, args []string) {
	logrus.Debug("Executing login command")

	var exitCode int

	if cfg.ClientCredentials.ClientId == "" || cfg.ClientCredentials.ClientSecret == "" {
		cfg.ReadClientCredentials()
	}

	if cfg.ClientCredentials.ClientId == "" || cfg.ClientCredentials.ClientSecret == "" {
		fmt.Println("Cannot execute login without client ID or client secret.")
		os.Exit(1)
	}

	cfg.Save(cfgFile)

	oauthClient := oauth.GetClient(cfg)
	expired, err := oauthClient.IsExpired()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if expired && oauthClient.HasRefreshToken() {
		exitCode = oauthClient.Refresh()
	} else if cfg.ClientCredentials.AccessToken == "" {
		exitCode = oauthClient.Get()
	} else {
		fmt.Println("You are already logged in to Foundries.io services.")
		os.Exit(0)
	}

	if exitCode == 0 {
		cfg.Save(cfgFile)
		fmt.Println("You are now logged in to Foundries.io services.")
	}

	os.Exit(exitCode)
}
