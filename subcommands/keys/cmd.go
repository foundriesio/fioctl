package keys

import (
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage keys in use by your factory fleet",
}

func NewCommand() *cobra.Command {
	return cmd
}
