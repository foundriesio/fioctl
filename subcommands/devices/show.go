package devices

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a specific device",
		Run:   doShow,
		Args:  cobra.ExactArgs(1),
	})
}

func doShow(cmd *cobra.Command, args []string) {
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
	if device.Group != nil {
		fmt.Printf("Group:\t\t%s\n", device.Group.Name)
	}
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
	if device.Network != nil {
		fmt.Println("Network Info:")
		fmt.Printf("\tHostname:\t%s\n", device.Network.Hostname)
		fmt.Printf("\tIP:\t\t%s\n", device.Network.Ipv4)
		fmt.Printf("\tMAC:\t\t%s\n", device.Network.MAC)
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
	if device.ActiveConfig != nil {
		fmt.Println("Active Config:")
		fmt.Println("\tCreated At: ", device.ActiveConfig.CreatedAt)
		fmt.Println("\tApplied At: ", device.ActiveConfig.AppliedAt)
		fmt.Println("\tReason:     ", device.ActiveConfig.Reason)
		fmt.Println("\tFiles:")
		for _, f := range device.ActiveConfig.Files {
			if len(f.OnChanged) == 0 {
				fmt.Printf("\t\t%s\n", f.Name)
			} else {
				fmt.Printf("\t\t%s - %v\n", f.Name, f.OnChanged)
			}
			if f.Unencrypted {
				for _, line := range strings.Split(f.Value, "\n") {
					fmt.Printf("\t\t | %s\n", line)
				}
			}
		}
	}
	if len(device.PublicKey) > 0 {
		fmt.Println()
		fmt.Print(device.PublicKey)
	}
}
