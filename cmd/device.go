package cmd

import (
	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:              "device",
	Short:            "Manage devices registered to a factory",
	PersistentPreRun: assertLogin,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
}
