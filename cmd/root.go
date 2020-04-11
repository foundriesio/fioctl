package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands/version"
)

var (
	cfgFile string
	api     *client.Api
	config  client.Config
	verbose bool
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print verbose logging")

	rootCmd.AddCommand(version.NewCommand())
}

func initViper(cmd *cobra.Command, args []string) {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
		fmt.Println("Error required flag \"factory\" not set")
		os.Exit(1)
	}
	config.Token = viper.GetString("token")
	url := os.Getenv("API_URL")
	if len(url) == 0 {
		url = "https://api.foundries.io"
	}
	ca := os.Getenv("CACERT")
	api = client.NewApiClient(url, config, ca)
}

func requireFactory(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("factory", "f", "", "Factory to list targets for")
	cmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
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
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Sprintf("Unexpected failure parsing configuration: %s", err))
	}
}
