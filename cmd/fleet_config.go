package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var fleetConfigCmd = &cobra.Command{
	Use:              "fleet-config file=content <file2=content ...>",
	Short:            "Create a new fleet wide configuration",
	PersistentPreRun: assertLogin,
	Run:              doFleetConfig,
	Args:             cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(fleetConfigCmd)
	fleetConfigCmd.Flags().StringVarP(&configReason, "reason", "m", "", "Add a message to store as the \"reason\" for this change")
	requireFactory(fleetConfigCmd)
}

func doFleetConfig(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Creating new fleet config for %s", factory)

	cfg := client.ConfigCreateRequest{Reason: configReason}
	for _, keyval := range args {
		parts := strings.SplitN(keyval, "=", 2)
		if len(parts) != 2 {
			fmt.Println("ERROR: Invalid file=content argument: ", keyval)
			os.Exit(1)
		}
		cfg.Files = append(cfg.Files, client.ConfigFile{Name: parts[0], Value: parts[1]})
	}

	if err := api.FleetCreateConfig(factory, cfg); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
