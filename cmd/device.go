package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deviceCmd = &cobra.Command{
	Use:              "devices",
	Aliases:          []string{"device"},
	Short:            "Manage devices registered to a factory",
	PersistentPreRun: assertLogin,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
	if err := viper.BindPFlags(deviceCmd.PersistentFlags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
