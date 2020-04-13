package devices

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

func init() {
	logConfigCmd := &cobra.Command{
		Use:   "log <device>",
		Short: "Changelog of device's configuration",
		Run:   doConfigLog,
		Args:  cobra.ExactArgs(1),
	}
	configCmd.AddCommand(logConfigCmd)
	logConfigCmd.Flags().IntVarP(&listLimit, "limit", "n", 0, "Limit the number of results displayed.")
}

func printConfigs(deviceConfigs []client.DeviceConfig) int {
	firstRow := color.New(color.FgYellow)
	for idx, cfg := range deviceConfigs {
		if idx != 0 {
			fmt.Println("")
		}
		firstRow.Printf("Created At:    %s\n", cfg.CreatedAt)
		fmt.Printf("Applied At:    %s\n", cfg.AppliedAt)
		fmt.Printf("Change Reason: %s\n", cfg.Reason)
		fmt.Println("Files:")
		for _, f := range cfg.Files {
			if len(f.OnChanged) == 0 {
				fmt.Printf("\t%s\n", f.Name)
			} else {
				fmt.Printf("\t%s - %v\n", f.Name, f.OnChanged)
			}
			if f.Unencrypted {
				for _, line := range strings.Split(f.Value, "\n") {
					fmt.Printf("\t | %s\n", line)
				}
			}
		}
		listLimit -= 1
		if listLimit == 0 {
			return 0
		}
	}
	return listLimit
}

func doConfigLog(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device config log")
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
		listLimit = printConfigs(dcl.Configs)
		if listLimit == 0 {
			break
		}
	}
}
