package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/internal/config"
)

var (
	cfgFile string
	api     *client.Api
	cfg     *config.Config
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
		cfgDir, err := homedir.Expand("~/.config")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cfgFile = filepath.Join(cfgDir, "fioctl.yaml")
		// Search config in home directory with name "fioctl" (without extension).
		viper.AddConfigPath(cfgDir)
		viper.SetConfigName("fioctl")
	}

	viper.SetEnvPrefix("FIOCTL")
	viper.AutomaticEnv()
	viper.ReadInConfig()
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if len(viper.GetString("token")) == 0 {
		rootCmd.MarkPersistentFlagRequired("token")
	}
	if len(viper.GetString("factory")) == 0 {
		targetsCmd.MarkPersistentFlagRequired("factory")
	}

	api = client.NewApiClient("https://api.foundries.io", viper.GetString("token"))
	cfg = config.Load(cfgFile)
}
