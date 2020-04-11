package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Commit string

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information of this tool.",
		Run:   doVersion,
	}
}

func doVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Commit)
}
