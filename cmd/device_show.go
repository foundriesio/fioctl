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
	fmt.Printf("UUID:\t\t%s\n", device.Uuid)
	fmt.Printf("Owner:\t\t%s\n", device.Owner)
	fmt.Printf("Factory:\t%s\n", device.Factory)
	fmt.Printf("Up to date:\t%v\n", device.UpToDate)
	fmt.Printf("Target:\t\t%s / sha256(%s)\n", device.TargetName, device.OstreeHash)
	fmt.Printf("Ostree Hash:\t%s\n", device.OstreeHash)
	fmt.Printf("Created:\t%s\n", device.CreatedAt)
	fmt.Printf("Last Seen:\t%s\n", device.LastSeen)
	if len(device.Tags) > 0 {
		fmt.Printf("Tags:\t\t%s\n", strings.Join(device.Tags, ","))
	}
	if len(device.DockerApps) > 0 {
		fmt.Printf("Docker Apps:\t%s\n", strings.Join(device.DockerApps, ","))
	}
	if len(device.Status) > 0 {
		fmt.Printf("Status:\t\t%s\n", device.Status)
	}
	if len(device.CurrentUpdate) > 0 {
		fmt.Printf("Update Id:\t%s\n", device.CurrentUpdate)
	}
	if device.Hardware != nil {
		b, err := json.MarshalIndent(device.Hardware, "\t", "  ")
		if err != nil {
			fmt.Println("Unable to marshall hardware info: ", err)
		}
		fmt.Printf("Hardware Info:\n\t")
		os.Stdout.Write(b)
		fmt.Println("")
	}
}
