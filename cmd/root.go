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
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Debug("Config file not found")
		} else {
			// Config file was found but another error was produced
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	}
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if len(viper.GetString("token")) == 0 {
		if err := rootCmd.MarkPersistentFlagRequired("token"); err != nil {
			panic(fmt.Sprintf("Unexpected failure in viper arg setup: %s",  err))
		}
	}
	if len(viper.GetString("factory")) == 0 {
		if err := rootCmd.MarkPersistentFlagRequired("factory"); err != nil {
			panic(fmt.Sprintf("Unexpected failure in viper arg setup: %s",  err))
		}
	}

	api = client.NewApiClient("https://api.foundries.io", viper.GetString("token"))
}
