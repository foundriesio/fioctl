package devices

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "rename <current name> <new name>",
		Short: "Rename a device",
		Run:   doRename,
		Args:  cobra.ExactArgs(2),
	})
}

func doRename(cmd *cobra.Command, args []string) {
	logrus.Debugf("Renaming %s -> %s", args[0], args[1])

	if err := api.DeviceRename(args[0], args[1]); err != nil {
		fmt.Printf("failed\n%s", err)
		os.Exit(1)
	}
}
