package cmd

import (
	"github.com/spf13/cobra"
)

var targetsCmd = &cobra.Command{
	Use:              "targets",
	Short:            "Manage factory's TUF targets",
	PersistentPreRun: assertLogin,
}

func init() {
	rootCmd.AddCommand(targetsCmd)
	requireFactory(targetsCmd)
}
