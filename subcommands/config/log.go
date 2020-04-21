package config

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Changelog of configuration",
		Run:   doConfigLog,
	}
	cmd.AddCommand(logCmd)
	logCmd.Flags().IntVarP(&listLimit, "limit", "n", 0, "Limit the number of results displayed.")
}

func doConfigLog(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing config history for %s", factory)
	var dcl *client.DeviceConfigList
	for {
		var err error
		if dcl == nil {
			dcl, err = api.FactoryListConfig(factory)
		} else {
			if dcl.Next != nil {
				dcl, err = api.FactoryListConfigCont(*dcl.Next)
			} else {
				break
			}
		}
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		listLimit = subcommands.PrintConfigs(dcl.Configs, listLimit)
		if listLimit == 0 {
			break
		}
	}
}
