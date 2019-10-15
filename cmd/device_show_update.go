package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var deviceShowUpdateCmd = &cobra.Command{
	Use:   "show-update [name] [update-id]",
	Short: "Show details of a specific device update",
	Run:   doDeviceShowUpdate,
	Args:  cobra.ExactArgs(2),
}

func init() {
	deviceCmd.AddCommand(deviceShowUpdateCmd)
}

func doDeviceShowUpdate(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device update")
	events, err := api.DeviceUpdateEvents(args[0], args[1])
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
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
	}
}
