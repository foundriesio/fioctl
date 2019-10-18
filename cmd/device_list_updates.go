package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var deviceUpdatesCmd = &cobra.Command{
	Use:   "list-updates <device>",
	Short: "List the device's update history",
	Run:   doDeviceUpdates,
	Args:  cobra.ExactArgs(1),
}

func init() {
	deviceCmd.AddCommand(deviceUpdatesCmd)
}

func doDeviceUpdates(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device updates")

	fmt.Printf("Update History:\n")
	var ul *client.UpdateList
	for {
		var err error
		if ul == nil {
			ul, err = api.DeviceListUpdates(args[0])
		} else {
			if ul.Next != nil {
				ul, err = api.DeviceListUpdatesCont(*ul.Next)
			} else {
				break
			}
		}
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		for idx, update := range ul.Updates {
			if idx > 0 {
				fmt.Println("")
			}
			fmt.Printf("Id:      %s\n", update.CorrelationId)
			fmt.Printf("Time:    %s\n", update.Time)
			fmt.Printf("Target:  %s\n", update.Target)
			fmt.Printf("Version: %s\n", update.Version)
		}
	}
}
