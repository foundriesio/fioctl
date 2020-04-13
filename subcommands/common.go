package subcommands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var (
	Config client.Config
)

func RequireFactory(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("factory", "f", "", "Factory to list targets for")
	cmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
}

func Login(cmd *cobra.Command) *client.Api {
	ca := os.Getenv("CACERT")
	initViper(cmd)
	if len(Config.Token) > 0 {
		return client.NewApiClient("https://api.foundries.io", Config, ca)
	}

	if len(Config.ClientCredentials.ClientId) == 0 {
		fmt.Println("ERROR: Please run: \"fioctl login\" first")
		os.Exit(1)
	}
	creds := client.NewClientCredentials(Config.ClientCredentials)

	expired, err := creds.IsExpired()
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}

	if !expired && len(creds.Config.AccessToken) > 0 {
		return client.NewApiClient("https://api.foundries.io", Config, ca)
	}

	if len(creds.Config.AccessToken) == 0 {
		if err := creds.Get(); err != nil {
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	} else if creds.HasRefreshToken() {
		if err := creds.Refresh(); err != nil {
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("ERROR: Missing refresh token")
		os.Exit(1)
	}
	SaveOauthConfig(creds.Config)
	return client.NewApiClient("https://api.foundries.io", Config, ca)
}

func SaveOauthConfig(c client.OAuthConfig) {
	viper.Set("clientcredentials.client_id", c.ClientId)
	viper.Set("clientcredentials.client_secret", c.ClientSecret)

	viper.Set("clientcredentials.access_token", c.AccessToken)
	viper.Set("clientcredentials.refresh_token", c.RefreshToken)
	viper.Set("clientcredentials.token_type", c.TokenType)
	viper.Set("clientcredentials.expires_in", c.ExpiresIn)
	viper.Set("clientcredentials.created", c.Created)

	if err := viper.WriteConfig(); err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}
}

func initViper(cmd *cobra.Command) {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
		fmt.Println("Error required flag \"factory\" not set")
		os.Exit(1)
	}
	Config.Token = viper.GetString("token")
	url := os.Getenv("API_URL")
	if len(url) == 0 {
		url = "https://api.foundries.io"
	}
}
