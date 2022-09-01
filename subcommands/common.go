package subcommands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/cheynewallace/tabby"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/shurcooL/go/indentwriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands/version"
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
	DieNotNil(viper.BindPFlags(cmd.Flags()))
	Config.Token = viper.GetString("token")
	url := os.Getenv("API_URL")
	if len(url) == 0 {
		url = "https://api.foundries.io"
	}
	if len(Config.Token) > 0 {
		if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
			DieNotNil(fmt.Errorf("Required flag \"factory\" not set"))
		}
		return client.NewApiClient(url, Config, ca, version.Commit)
	}

	if len(Config.ClientCredentials.ClientId) == 0 {
		DieNotNil(fmt.Errorf("Please run: \"fioctl login\" first"))
	}
	if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
		DieNotNil(fmt.Errorf("Required flag \"factory\" not set"))
	}
	creds := client.NewClientCredentials(Config.ClientCredentials)

	expired, err := creds.IsExpired()
	DieNotNil(err)

	if !expired && len(creds.Config.AccessToken) > 0 {
		return client.NewApiClient(url, Config, ca, version.Commit)
	}

	if len(creds.Config.AccessToken) == 0 {
		DieNotNil(creds.Get())
	} else if creds.HasRefreshToken() {
		DieNotNil(creds.Refresh())
	} else {
		DieNotNil(fmt.Errorf("Missing refresh token"))
	}
	SaveOauthConfig(creds.Config)
	Config.ClientCredentials = creds.Config
	return client.NewApiClient(url, Config, ca, version.Commit)
}

func SaveOauthConfig(c client.OAuthConfig) {
	viper.Set("clientcredentials.client_id", c.ClientId)
	viper.Set("clientcredentials.client_secret", c.ClientSecret)

	viper.Set("clientcredentials.access_token", c.AccessToken)
	viper.Set("clientcredentials.refresh_token", c.RefreshToken)
	viper.Set("clientcredentials.token_type", c.TokenType)
	viper.Set("clientcredentials.expires_in", c.ExpiresIn)
	viper.Set("clientcredentials.created", c.Created)

	// viper.WriteConfig isn't so great for this. It doesn't just write
	// these values but any other flags that were present when this runs.
	// This gets run automatically when "logging in". So you sometimes
	// accidentally write CLI flags viper finds to the file, that you
	// don't intend to be saved. So we do it the hard way:
	name := viper.ConfigFileUsed()
	if len(name) == 0 {
		logrus.Debug("Guessing config file from path")
		path, err := homedir.Expand("~/.config")
		DieNotNil(err)
		name = filepath.Join(path, "fioctl.yaml")
	}
	// Try to read in config
	cfg := make(map[string]interface{})
	buf, err := os.ReadFile(name)
	if err == nil {
		DieNotNil(yaml.Unmarshal(buf, &cfg), "Unable unmarshal configuration:")
	}
	val := viper.Get("clientcredentials")
	cfg["clientcredentials"] = val
	if len(c.DefaultOrg) > 0 {
		cfg["factory"] = c.DefaultOrg
	}
	buf, err = yaml.Marshal(cfg)
	DieNotNil(err, "Unable to marshall oauth config:")
	DieNotNil(os.WriteFile(name, buf, os.FileMode(0644)), "Unable to update config: ")
}

func DieNotNil(err error, message ...string) {
	if err != nil {
		parts := []interface{}{"ERROR:"}
		for _, p := range message {
			parts = append(parts, p)
		}
		parts = append(parts, err)
		fmt.Println(parts...)
		os.Exit(1)
	}
}

func Tabby(indent int, columns ...interface{}) *tabby.Tabby {
	var out io.Writer = os.Stdout
	if indent > 0 {
		out = indentwriter.New(out, indent)
	}
	tab := tabby.NewCustom(tabwriter.NewWriter(out, 0, 0, 2, ' ', 0))
	if len(columns) > 0 {
		tab.AddHeader(columns...)
	}
	return tab
}
