package devices

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(&cobra.Command{
		Use:   "delete <device> <file>",
		Short: "Delete file from the current configuration",
		Run:   doConfigDelete,
		Args:  cobra.MinimumNArgs(2),
	})
}

func doConfigDelete(cmd *cobra.Command, args []string) {
	logrus.Debug("Deleting file from device config")

	if err := api.DeviceDeleteConfig(args[0], args[1]); err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}

}
