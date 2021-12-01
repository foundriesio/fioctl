package devices

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	updatesCmd.AddCommand(&cobra.Command{
		Use:    "show <name> <update-id>",
		Short:  "[DEPRECATED] Please use: fioctl devices updates <device> <update-id>",
		Hidden: true,
		Run:    doShowUpdate,
		Args:   cobra.ExactArgs(2),
	})
}

func doShowUpdate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debug("Showing device update")
	events, err := api.DeviceUpdateEvents(factory, args[0], args[1])
	subcommands.DieNotNil(err)
	for _, event := range events {
		fmt.Printf("%s : %s(%s)", event.Time, event.Type.Id, event.Detail.TargetName)
		if event.Detail.Success != nil {
			if *event.Detail.Success {
				fmt.Println(" -> Succeed")
			} else {
				fmt.Println(" -> Failed!")
			}
		} else {
			fmt.Println()
		}
		if len(event.Detail.Details) > 0 {
			fmt.Println(" Details:")
			indented := " | " + strings.Replace(event.Detail.Details, "\n", "\n | ", -1)
			fmt.Println(indented)
		}
	}
}
