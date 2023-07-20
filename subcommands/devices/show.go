package devices

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	factory := viper.GetString("factory")
	logrus.Debug("Showing device")
	device, err := api.DeviceGet(factory, args[0])
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
	fmt.Printf("Created At:\t%s\n", device.ChangeMeta.CreatedAt)
	if len(device.ChangeMeta.CreatedBy) > 0 {
		fmt.Printf("Created By:\t%s\n", device.ChangeMeta.CreatedBy)
	}
	if len(device.ChangeMeta.UpdatedAt) > 0 {
		fmt.Printf("Updated At:\t%s\n", device.ChangeMeta.UpdatedAt)
	}
	if len(device.ChangeMeta.UpdatedBy) > 0 {
		fmt.Printf("Updated By:\t%s\n", device.ChangeMeta.UpdatedBy)
	}
	fmt.Printf("Last Seen:\t%s\n", device.LastSeen)
	if len(device.Tag) > 0 {
		fmt.Printf("Tag:\t\t%s\n", device.Tag)
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
	if device.AppsState != nil && device.AppsState.Apps != nil {
		var healthyApps []string
		var unhealthyApps []string
		for a, s := range device.AppsState.Apps {
			if s.State == "healthy" {
				healthyApps = append(healthyApps, a)
			} else {
				unhealthyApps = append(unhealthyApps, a)
			}
		}
		fmt.Printf("Healthy Apps:\t%s\n", strings.Join(healthyApps, ","))
		fmt.Printf("Unhealthy Apps:\t%s\n", strings.Join(unhealthyApps, ","))
	}
	if device.Network != nil {
		fmt.Println("Network Info:")
		fmt.Printf("\tHostname:\t%s\n", device.Network.Hostname)
		fmt.Printf("\tIP:\t\t%s\n", device.Network.Ipv4)
		fmt.Printf("\tMAC:\t\t%s\n", device.Network.MAC)
	}
	if len(device.Secondaries) > 0 {
		fmt.Println("Secondary ECUs:")
		for _, ecu := range device.Secondaries {
			fmt.Printf("\t%s / Target(%s)", ecu.Serial, ecu.TargetName)
			if len(ecu.HardwareId) > 0 {
				fmt.Printf(" Hardware ID(%s)", ecu.HardwareId)
			}
			fmt.Println()
		}
	}
	if device.Hardware != nil {
		b, err := subcommands.MarshalIndent(device.Hardware, "\t", "  ")
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
		if len(device.ActiveConfig.CreatedBy) > 0 {
			user, err := api.UserAccessDetails(factory, device.ActiveConfig.CreatedBy)
			if err != nil {
				device.ActiveConfig.CreatedBy = fmt.Sprintf("%s / ?", device.ActiveConfig.CreatedBy)
			} else {
				device.ActiveConfig.CreatedBy = fmt.Sprintf("%s / %s", user.PolisId, user.Name)
			}
		}
		subcommands.PrintConfig(device.ActiveConfig, true, false, "\t")
	}
	if len(device.PublicKey) > 0 {
		fmt.Println()
		fmt.Print(device.PublicKey)
	}
}
