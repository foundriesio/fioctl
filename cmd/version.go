package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Commit string
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information of this tool.",
	Run:   doVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func doVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Commit)
}
