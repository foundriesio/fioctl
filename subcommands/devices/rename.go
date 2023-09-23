package devices

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	factory := viper.GetString("factory")
	logrus.Debugf("Renaming %s -> %s", args[0], args[1])

	if err := api.DeviceRename(factory, args[0], args[1]); err != nil {
		color.Red(fmt.Sprintf("failed\n%s", err))
		os.Exit(1)
	}
}
