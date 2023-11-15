package login

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	refreshToken bool
	authURL      string
	insecure     bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Access Foundries.io services with your client credentials",
		Run:   doLogin,
	}
	cmd.Flags().BoolVarP(&refreshToken, "refresh-access-token", "", false, "Refresh your current oauth2 access token. This is used when a token's scopes have been updated in app.foundries.io")
	cmd.Flags().StringVarP(&authURL, "oauth-url", "", client.OauthURL, "OAuth URL to authenticate with")
	cmd.Flags().BoolVarP(&insecure, "insecure-ssl", "", false, "Ignore TLS certificates from API servers.")
	_ = cmd.Flags().MarkHidden("insecure-ssl")
	return cmd
}

func doLogin(cmd *cobra.Command, args []string) {
	logrus.Debug("Executing login command")

	if refreshToken {
		creds := client.NewClientCredentials(subcommands.Config.ClientCredentials)
		// Change ExpiresIn to basically "now". This will cause fioctl to
		// get a new token with fresh scopes
		creds.Config.ExpiresIn = 1
		subcommands.SaveOauthConfig(creds.Config)
		return
	}
	if !cmd.Flag("oauth-url").Changed {
		authURL = subcommands.Config.ClientCredentials.URL
	}
	if len(authURL) == 0 {
		authURL = client.OauthURL
	}
	u, err := url.Parse(authURL)
	subcommands.DieNotNil(err)
	subcommands.Config.ClientCredentials.URL = authURL

	if authURL != client.OauthURL {
		apiUrl := "https://" + strings.Replace(u.Host, "app", "api", 1)
		logrus.Debugf("Configuring REST API based on oauth url to: %s", apiUrl)
		viper.Set("server.url", apiUrl)
	}

	creds := client.NewClientCredentials(subcommands.Config.ClientCredentials)
	if creds.Config.ClientId == "" || creds.Config.ClientSecret == "" {
		credsUrl := fmt.Sprintf("https://%s/settings/credentials/", u.Host)
		creds.Config.ClientId, creds.Config.ClientSecret = promptForCreds(credsUrl)
	}

	if insecure {
		logrus.Debug("Configuring for insecure TLS connections")
		viper.Set("server.insecure_skip_verify", true)
		creds.InsecureSSL = true
	}

	if creds.Config.ClientId == "" || creds.Config.ClientSecret == "" {
		fmt.Println("Cannot execute login without client ID or client secret.")
		os.Exit(1)
	}

	expired, err := creds.IsExpired()
	subcommands.DieNotNil(err)

	if expired && creds.HasRefreshToken() {
		subcommands.DieNotNil(creds.Refresh())
	} else if creds.Config.AccessToken == "" {
		subcommands.DieNotNil(creds.Get())
	} else {
		fmt.Println("You are already logged in to Foundries.io services.")
		os.Exit(0)
	}

	subcommands.SaveOauthConfig(creds.Config)
	fmt.Println("You are now logged in to Foundries.io services.")
}

func promptForCreds(credsUrl string) (string, string) {
	logrus.Debug("Reading client ID/secret from stdin")

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Please visit:\n\n")
	fmt.Printf("  %s\n\n", credsUrl)
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
