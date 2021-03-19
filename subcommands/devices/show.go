package devices

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	showHWInfo bool
	showAkToml bool
)

func init() {
	showCmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a specific device",
		Run:   doShow,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(showCmd)
	showCmd.Flags().BoolVarP(&showHWInfo, "hwinfo", "i", false, "Show HW Information")
	showCmd.Flags().BoolVarP(&showAkToml, "aktoml", "", false, "Show aktualizr-lite toml config")
}

func doShow(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device")
	device, err := api.DeviceGet(args[0])
	subcommands.DieNotNil(err)

	fmt.Printf("UUID:\t\t%s\n", device.Uuid)
	fmt.Printf("Owner:\t\t%s\n", device.Owner)
	fmt.Printf("Factory:\t%s\n", device.Factory)
	if device.Group != nil {
		fmt.Printf("Group:\t\t%s\n", device.Group.Name)
	}
	var waveSuffix string
	if device.IsWave {
		waveSuffix = " (in wave)"
	}
	fmt.Printf("Production:\t%v%s\n", device.IsProd, waveSuffix)
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
		if showHWInfo {
			fmt.Printf("Hardware Info:\n\t")
			os.Stdout.Write(b)
			fmt.Println("")
		} else {
			fmt.Printf("Hardware Info: (hidden, use --hwinfo)\n")
		}
	}
	if len(device.AktualizrToml) > 0 {
		if showAkToml {
			for _, line := range strings.Split(device.AktualizrToml, "\n") {
				fmt.Printf("\t| %s\n", line)
			}
		} else {
			fmt.Println("Aktualizr config: (hidden, use --aktoml)")
		}

	}
	if device.ActiveConfig != nil {
		fmt.Println("Active Config:")
		subcommands.PrintConfig(device.ActiveConfig, true, false, "\t")
	}
	if len(device.PublicKey) > 0 {
		fmt.Println()
		fmt.Print(device.PublicKey)
	}
}
