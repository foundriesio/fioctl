package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
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
advantage of this, the "--raw" flag must be used. eg::

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
Instead of using ./tmp.json, the command can take a "-" and will read the
content from STDIN instead of a file.

Use a -g or --group parameter to create a device group wide configuration instead.
`,
		Run:  doConfigSet,
		Args: cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(setCmd)
	setCmd.Flags().StringP("group", "g", "", "Device group to use")
	setCmd.Flags().StringP("reason", "m", "", "Add a message to store as the \"reason\" for this change")
	setCmd.Flags().BoolP("raw", "", false, "Use raw configuration file")
	setCmd.Flags().BoolP("create", "", false, "Replace the whole config with these values. Default is to merge these values in with the existing config values")
}

func doConfigSet(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	group, _ := cmd.Flags().GetString("group")
	reason, _ := cmd.Flags().GetString("reason")
	isRaw, _ := cmd.Flags().GetBool("raw")
	shouldCreate, _ := cmd.Flags().GetBool("create")
	opts := subcommands.SetConfigOptions{FileArgs: args, Reason: reason, IsRawFile: isRaw}

	if group == "" {
		logrus.Debugf("Creating new config for %s", factory)
		opts.SetFunc = func(cfg client.ConfigCreateRequest) error {
			if shouldCreate {
				return api.FactoryCreateConfig(factory, cfg)
			} else {
				return api.FactoryPatchConfig(factory, cfg, false)
			}
		}
	} else {
		logrus.Debugf("Creating new config for %s group %s", factory, group)
		opts.SetFunc = func(cfg client.ConfigCreateRequest) error {
			if shouldCreate {
				return api.GroupCreateConfig(factory, group, cfg)
			} else {
				return api.GroupPatchConfig(factory, group, cfg, false)
			}
		}
	}
	subcommands.SetConfig(&opts)
}
