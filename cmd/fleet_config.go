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
	Use:   "fleet-config file=content <file2=content ...>",
	Short: "Create a new fleet wide configuration",
	Long: `
Creates a fleet wide configuration. The fioconfig daemon running on
each device will then be able to grab the latest version of the
feelt configuration and the device's configuration and apply it.

Basic use can be done with command line arguments. eg:

  fioctl fleet-config \
    npmtok="root" \
    githubtok="1234"

The configuration format also allows specifying what command to
run after a configuration file is updated on the device. To take
advantage of this, the "--raw" flag must be used. eg:

  cat >tmp.json <<EOF
  {
    "reason": "I want to use the on-changed attribute",
    "files": [
      {
        "name": "npmtok",
	"value": "root",
	"on-changed": ["/usr/bin/touch", "/tmp/npmtok-changed"]
      },
      {
        "name": "githubok",
	"value": "1234",
	"on-changed": ["/usr/sbin/reboot"]
      }
    ]
  }
  > EOF
  fioctl fleet-config --raw ./tmp.json

fioctl will read in tmp.json and upload it to the OTA server.
`,
	PersistentPreRun: assertLogin,
	Run:              doFleetConfig,
	Args:             cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(fleetConfigCmd)
	fleetConfigCmd.Flags().StringVarP(&configReason, "reason", "m", "", "Add a message to store as the \"reason\" for this change")
	fleetConfigCmd.Flags().BoolVarP(&configRaw, "raw", "", false, "Use raw configuration file")
	requireFactory(fleetConfigCmd)
}

func doFleetConfig(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Creating new fleet config for %s", factory)

	cfg := client.ConfigCreateRequest{Reason: configReason}
	if configRaw {
		loadConfig(args[0], &cfg)
	} else {
		for _, keyval := range args {
			parts := strings.SplitN(keyval, "=", 2)
			if len(parts) != 2 {
				fmt.Println("ERROR: Invalid file=content argument: ", keyval)
				os.Exit(1)
			}
			cfg.Files = append(cfg.Files, client.ConfigFile{Name: parts[0], Value: parts[1]})
		}
	}

	if err := api.FleetCreateConfig(factory, cfg); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
