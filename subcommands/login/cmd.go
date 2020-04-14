package login

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Access Foundries.io services with your client credentials",
		Run:   doLogin,
	}
}

func doLogin(cmd *cobra.Command, args []string) {
	logrus.Debug("Executing login command")

	creds := client.NewClientCredentials(subcommands.Config.ClientCredentials)
	if creds.Config.ClientId == "" || creds.Config.ClientSecret == "" {
		creds.Config.ClientId, creds.Config.ClientSecret = promptForCreds()
		subcommands.SaveOauthConfig(creds.Config)
	}

	if creds.Config.ClientId == "" || creds.Config.ClientSecret == "" {
		fmt.Println("Cannot execute login without client ID or client secret.")
		os.Exit(1)
	}

	expired, err := creds.IsExpired()
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}

	if expired && creds.HasRefreshToken() {
		if err := creds.Refresh(); err != nil {
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	} else if creds.Config.AccessToken == "" {
		if err := creds.Get(); err != nil {
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("You are already logged in to Foundries.io services.")
		os.Exit(0)
	}

	subcommands.SaveOauthConfig(creds.Config)
	fmt.Println("You are now logged in to Foundries.io services.")
}

func promptForCreds() (string, string) {
	logrus.Debug("Reading client ID/secret from stdin")

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Please visit:\n\n")
	fmt.Print("  https://app.foundries.io/settings/tokens/\n\n")
	fmt.Print("and create a new \"Application Credential\" to provide inputs below.\n\n")
	fmt.Print("Client ID: ")
	scanner.Scan()
	clientId := strings.Trim(scanner.Text(), " ")

	fmt.Print("Client secret: ")
	scanner.Scan()
	clientSecret := strings.Trim(scanner.Text(), " ")

	if clientId == "" || clientSecret == "" {
		fmt.Println("Client ID and client credentials are both required.")
		os.Exit(1)
	}

	return clientId, clientSecret
}
