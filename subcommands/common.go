package subcommands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

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
	url := os.Getenv("API_URL")
	if len(url) == 0 {
		url = "https://api.foundries.io"
	}
	if len(Config.Token) > 0 {
		return client.NewApiClient(url, Config, ca)
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
	Config.ClientCredentials = creds.Config
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

	// viper.WriteConfig isn't so great for this. It doesn't just write
	// these values but any other flags that were present when this runs.
	// This gets run automatically when "logging in". So you sometimes
	// accidentally write CLI flags viper finds to the file, that you
	// don't intend to be saved. So we do it the hard way:
	name := viper.ConfigFileUsed()
	if len(name) == 0 {
		logrus.Debug("Guessing config file from path")
		path, err := homedir.Expand("~/.config")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		name = filepath.Join(path, "fioctl.yaml")
	}
	// Try to read in config
	cfg := make(map[string]interface{})
	buf, err := ioutil.ReadFile(name)
	if err == nil {
		if err := yaml.Unmarshal(buf, &cfg); err != nil {
			fmt.Println("Unable unmarshal configuration:", err)
			os.Exit(1)
		}
	}
	val := viper.Get("clientcredentials")
	cfg["clientcredentials"] = val
	buf, err = yaml.Marshal(cfg)
	if err != nil {
		fmt.Println("Unable to marshall oauth config: ", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(name, buf, os.FileMode(0644)); err != nil {
		fmt.Println("Unable to update config: ", err)
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
}

func PrintConfigs(deviceConfigs []client.DeviceConfig, listLimit int) int {
	firstRow := color.New(color.FgYellow)
	for idx, cfg := range deviceConfigs {
		if idx != 0 {
			fmt.Println("")
		}
		firstRow.Printf("Created At:    %s\n", cfg.CreatedAt)
		fmt.Printf("Applied At:    %s\n", cfg.AppliedAt)
		fmt.Printf("Change Reason: %s\n", cfg.Reason)
		fmt.Println("Files:")
		for _, f := range cfg.Files {
			if len(f.OnChanged) == 0 {
				fmt.Printf("\t%s\n", f.Name)
			} else {
				fmt.Printf("\t%s - %v\n", f.Name, f.OnChanged)
			}
			if f.Unencrypted {
				for _, line := range strings.Split(f.Value, "\n") {
					fmt.Printf("\t | %s\n", line)
				}
			}
		}
		listLimit -= 1
		if listLimit == 0 {
			return 0
		}
	}
	return listLimit
}
