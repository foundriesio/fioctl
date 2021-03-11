package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	cfgcmd "github.com/foundriesio/fioctl/subcommands/config"
	"github.com/foundriesio/fioctl/subcommands/devices"
	"github.com/foundriesio/fioctl/subcommands/keys"
	"github.com/foundriesio/fioctl/subcommands/login"
	"github.com/foundriesio/fioctl/subcommands/secrets"
	"github.com/foundriesio/fioctl/subcommands/status"
	"github.com/foundriesio/fioctl/subcommands/targets"
	"github.com/foundriesio/fioctl/subcommands/users"
	"github.com/foundriesio/fioctl/subcommands/version"
	"github.com/foundriesio/fioctl/subcommands/waves"
)

var (
	cfgFile string
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

	rootCmd.AddCommand(cfgcmd.NewCommand())
	rootCmd.AddCommand(devices.NewCommand())
	rootCmd.AddCommand(keys.NewCommand())
	rootCmd.AddCommand(login.NewCommand())
	rootCmd.AddCommand(users.NewCommand())
	rootCmd.AddCommand(secrets.NewCommand())
	rootCmd.AddCommand(status.NewCommand())
	rootCmd.AddCommand(targets.NewCommand())
	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(waves.NewCommand())
	rootCmd.AddCommand(subcommands.GetCommand)

	rootCmd.AddCommand(docsCmd)
}

func getConfigDir() string {
	config, err := homedir.Expand("~/.config")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if err := os.Mkdir(config, 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return config
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name "fioctl" (without extension).
		viper.AddConfigPath(getConfigDir())
		viper.SetConfigName("fioctl")
		viper.SetConfigType("yaml")
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
	subcommands.Config = config
}
