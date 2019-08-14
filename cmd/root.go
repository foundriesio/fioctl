package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var (
	cfgFile string
	api     *client.Api
)

var rootCmd = &cobra.Command{
	Use:   "fioctl",
	Short: "Manage Foundries Factories",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/fioctl.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Print verbose logging")

	rootCmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
	rootCmd.MarkPersistentFlagRequired("token")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		config, err := homedir.Expand("~/.config")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name "fioctl" (without extension).
		viper.AddConfigPath(config)
		viper.SetConfigName("fioctl")
	}

	viper.SetEnvPrefix("FIOCTL")
	viper.AutomaticEnv()
	viper.ReadInConfig()
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	api = client.NewApiClient("https://api.foundries.io", viper.GetString("token"))
}
