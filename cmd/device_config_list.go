package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var deviceConfigsCmd = &cobra.Command{
	Use:   "list-config <device>",
	Short: "List the device's config history",
	Run:   doDeviceConfigs,
	Args:  cobra.ExactArgs(1),
}

var deviceConfigsLimit int

func init() {
	deviceCmd.AddCommand(deviceConfigsCmd)
	deviceConfigsCmd.Flags().IntVarP(&deviceConfigsLimit, "limit", "n", 0, "Limit the number of results displayed.")
}

func doDeviceConfigs(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device config history")
	var dcl *client.DeviceConfigList
	for {
		var err error
		if dcl == nil {
			dcl, err = api.DeviceListConfig(args[0])
		} else {
			if dcl.Next != nil {
				dcl, err = api.DeviceListConfigCont(*dcl.Next)
			} else {
				break
			}
		}
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		for idx, cfg := range dcl.Configs {
			if idx != 0 {
				fmt.Println("")
			}
			fmt.Printf("Created At:    %s\n", cfg.CreatedAt)
			fmt.Printf("Applied At:    %s\n", cfg.AppliedAt)
			fmt.Printf("Change Reason: %s\n", cfg.Reason)
			fmt.Println("Files:")
			for _, f := range cfg.Files {
				if len(f.OnChanged) == 0 {
					fmt.Printf("\t%s\n", f.Name)
				} else {
					fmt.Printf("\t%s - %v\n", f.Name, f.OnChanged)
				}
			}

			deviceConfigsLimit -= 1
			if deviceConfigsLimit == 0 {
				break
			}
		}
		if deviceConfigsLimit == 0 {
			break
		}
	}
}
