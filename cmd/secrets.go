package cmd

import (
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:              "secrets",
	Short:            "Manage secret crendentials configured in a factory",
	PersistentPreRun: assertLogin,
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	requireFactory(secretsCmd)
}
