package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	configUpdatesCmd := &cobra.Command{
		Use:   "updates",
		Short: "Configure aktualizr-lite settings for how updates are applied factory wide to devices",
		Run:   doConfigUpdates,
		Long: `View or change factory wide configuration parameters used by aktualizr-lite for updating devices.
When run with no options, this command will print out how the factory is
currently configured. Use the --group parameter to view or change a device 
group wide configuration instead.`,
		Example: `
  # Make devices start taking updates from Targets tagged with "devel":
  fioctl config updates --tag devel

  # Set the docker apps that devices will run:
  fioctl config updates --apps shellhttpd

  # Set the docker apps and the tag for devices:
  fioctl config updates --apps shellhttpd --tag master`,
	}
	cmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringP("group", "g", "", "Device group to use")
	configUpdatesCmd.Flags().StringP("tag", "", "", "Tag for devices to follow")
	configUpdatesCmd.Flags().StringP("tags", "", "", "Tag for devices to follow")
	configUpdatesCmd.Flags().StringP("apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolP("dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolP("force", "", false, "DANGER: For a config on a device that might result in corruption")

	_ = configUpdatesCmd.Flags().MarkHidden("tags")        // assign for go linter
}

func doConfigUpdates(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	group, _ := cmd.Flags().GetString("group")
	updateApps, _ := cmd.Flags().GetString("apps")
	updateTag, _ := cmd.Flags().GetString("tag")
	if len(updateTag) == 0 {
		// check the old, deprecated "tags" option
		updateTag, _ = cmd.Flags().GetString("tags")
	}
	isDryRun, _ := cmd.Flags().GetBool("dryrun")
	isForced, _ := cmd.Flags().GetBool("force")

	opts := subcommands.SetUpdatesConfigOptions{
		UpdateApps: updateApps,
		UpdateTag:  updateTag,
		IsDryRun:   isDryRun,
		IsForced:   isForced,
	}

	if group == "" {
		logrus.Debugf("Configuring factory wide device updates for %s", factory)
		opts.ListFunc = func() (*client.DeviceConfigList, error) {
			return api.FactoryListConfig(factory)
		}
		opts.SetFunc = func(cfg client.ConfigCreateRequest, force bool) error {
			return api.FactoryPatchConfig(factory, cfg, force)
		}
	} else {
		logrus.Debugf("Configuring group wide device updates for %s group %s", factory, group)
		opts.ListFunc = func() (*client.DeviceConfigList, error) {
			return api.GroupListConfig(factory, group)
		}
		opts.SetFunc = func(cfg client.ConfigCreateRequest, force bool) error {
			return api.GroupPatchConfig(factory, group, cfg, force)
		}
	}
	subcommands.SetUpdatesConfig(&opts, "", nil)
}
