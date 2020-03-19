package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cheynewallace/tabby"
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
	t := tabby.New()
	t.AddHeader("CREATED", "APPLIED", "FILES", "REASON")
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
		for _, cfg := range dcl.Configs {
			t.AddLine(cfg.CreatedAt, cfg.AppliedAt, strings.Join(cfg.Files, ","), cfg.Reason)
			deviceConfigsLimit -= 1
			if deviceConfigsLimit == 0 {
				break
			}
		}
		if deviceConfigsLimit == 0 {
			break
		}
	}
	t.Print()
}
