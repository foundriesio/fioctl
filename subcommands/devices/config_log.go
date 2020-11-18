package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
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
		subcommands.DieNotNil(err)
		listLimit = subcommands.PrintConfigs(dcl.Configs, listLimit)
		if listLimit == 0 {
			break
		}
	}
}
