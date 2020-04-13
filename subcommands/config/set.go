package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var (
	configReason string
	configRaw    bool
	configCreate bool
)

func init() {
	setCmd := &cobra.Command{
		Use:   "set file=content <file2=content ...>",
		Short: "Create a new factory-wide configuration",
		Long: `Creates a factory wide configuration. The fioconfig daemon running on
each device will then be able to grab the latest version of the configuration
and the device's configuration and apply it.

Basic use can be done with command line arguments. eg:

  fioctl config set npmtok="root" githubtok="1234"

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
        "name": "A-Readable-Value",
        "value": "This won't be encrypted and will be visible from the API",
        "unencrypted": true
      },
      {
        "name": "githubok",
        "value": "1234"
      }
    ]
  }
  > EOF
  fioctl config set --raw ./tmp.json

fioctl will read in tmp.json and upload it to the OTA server.
`,
		Run:  doConfigSet,
		Args: cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(setCmd)
	setCmd.Flags().StringVarP(&configReason, "reason", "m", "", "Add a message to store as the \"reason\" for this change")
	setCmd.Flags().BoolVarP(&configRaw, "raw", "", false, "Use raw configuration file")
	setCmd.Flags().BoolVarP(&configCreate, "create", "", false, "Replace the whole config with these values. Default is to merge these values in with the existing config values")
}

func loadConfig(configFile string, cfg *client.ConfigCreateRequest) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("ERROR: Unable to read config file: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(content, cfg); err != nil {
		fmt.Printf("ERROR: Unable to parse config file: %v\n", err)
		os.Exit(1)
	}
}

func doConfigSet(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Creating new config for %s", factory)

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

	if configCreate {
		if err := api.FactoryCreateConfig(factory, cfg); err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		if err := api.FactoryPatchConfig(factory, cfg); err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
