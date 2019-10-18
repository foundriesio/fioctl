package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var deviceShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a specific device",
	Run:   doDeviceShow,
	Args:  cobra.ExactArgs(1),
}

func init() {
	deviceCmd.AddCommand(deviceShowCmd)
}

func doDeviceShow(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device")
	device, err := api.DeviceGet(args[0])
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("\tUUID:\t\t%s\n", device.Uuid)
	fmt.Printf("\tOwner:\t\t%s\n", device.Owner)
	fmt.Printf("\tFactory:\t%s\n", device.Factory)
	fmt.Printf("\tTarget:\t\t%s / sha256(%s)\n", device.TargetName, device.OstreeHash)
	fmt.Printf("\tOstree Hash:\t%s\n", device.OstreeHash)
	fmt.Printf("\tCreated:\t%s\n", device.CreatedAt)
	fmt.Printf("\tLast Seen:\t%s\n", device.LastSeen)
	if len(device.Tags) > 0 {
		fmt.Printf("\tTags:\t\t%s\n", strings.Join(device.Tags, ","))
	}
	if len(device.DockerApps) > 0 {
		fmt.Printf("\tDocker Apps:\t%s\n", strings.Join(device.DockerApps, ","))
	}
	if len(device.Status) > 0 {
		fmt.Printf("\tStatus:\t\t%s\n", device.Status)
	}
	if len(device.CurrentUpdate) > 0 {
		fmt.Printf("\tUpdate Id:\t%s\n", device.CurrentUpdate)
	}
	if device.Hardware != nil {
		b, err := json.MarshalIndent(device.Hardware, "\t\t", "  ")
		if err != nil {
			fmt.Println("Unable to marshall hardware info: ", err)
		}
		fmt.Printf("\tHardware Info:\n\t\t")
		os.Stdout.Write(b)
		fmt.Println("")
	}
	if len(device.Updates) > 0 {
		fmt.Printf("\tUpdate History:\n")
		for idx, update := range device.Updates {
			if idx > 0 {
				fmt.Printf("\t\t-------------------------------------------------\n\n")
			}
			fmt.Printf("\t\tId:      %s\n", update.CorrelationId)
			fmt.Printf("\t\tTime:    %s\n", update.Time)
			fmt.Printf("\t\tTarget:  %s\n", update.Target)
			fmt.Printf("\t\tVersion: %s\n", update.Version)
		}
	}
}
