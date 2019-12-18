package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var targetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "Manage factory's TUF targets",
}

func init() {
	rootCmd.AddCommand(targetsCmd)
	targetsCmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
	targetsCmd.PersistentFlags().StringP("factory", "f", "", "Factory to list targets for")

	if err := viper.BindPFlags(targetsCmd.PersistentFlags()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
