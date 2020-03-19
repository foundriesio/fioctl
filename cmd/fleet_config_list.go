package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var fleetConfigsCmd = &cobra.Command{
	Use:              "fleet-list-config",
	Short:            "List the fleet-wide config history for a factory",
	PersistentPreRun: assertLogin,
	Run:              doFleetConfigs,
}

func init() {
	rootCmd.AddCommand(fleetConfigsCmd)
	fleetConfigsCmd.Flags().IntVarP(&deviceConfigsLimit, "limit", "n", 0, "Limit the number of results displayed.")
	requireFactory(fleetConfigsCmd)
}

func doFleetConfigs(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing fleet history for %s", factory)
	var dcl *client.DeviceConfigList
	for {
		var err error
		if dcl == nil {
			dcl, err = api.FleetListConfig(factory)
		} else {
			if dcl.Next != nil {
				dcl, err = api.FleetListConfigCont(*dcl.Next)
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
