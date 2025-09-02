package devices

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func doShowUpdate(_ *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debug("Showing device update")
	d := api.DeviceApiByName(factory, args[0])
	events, err := d.UpdateEvents(args[1])
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
			indented := " | " + strings.ReplaceAll(event.Detail.Details, "\n", "\n | ")
			fmt.Println(indented)
		}
	}
}
