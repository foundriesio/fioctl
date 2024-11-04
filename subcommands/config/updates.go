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
		Short: "Configure aktualizr-lite settings for how updates are applied to a device group",
		Run:   doConfigUpdates,
		Long: `View or change configuration parameters used by aktualizr-lite for updating devices
in a device group. When run without options, prints out the current configuration.`,
		Example: `
  # Make devices start taking updates from Targets tagged with "devel":
  fioctl config updates --group beta --tag devel

  # Set the Compose apps that devices will run:
  fioctl config updates --group beta --apps shellhttpd

  # Set the Compose apps and the tag for devices:
  fioctl config updates --group beta --apps shellhttpd --tag master

  # There are two special characters: "," and "-".
  # Providing a "," sets the Compose apps to "none" for devices.
  # This will make the device run no apps:
  fioctl config updates --apps ,

  # Providing a "-" sets the Compose apps to "preset-apps" (all apps on most devices).
  # The system looks in the following locations to get the complete config:
       - /usr/lib/sota/conf.d/
       - /var/sota/sota.toml
       - /etc/sota/conf.d/
  fioctl config updates --apps -

  # Set the device tag to a "preset-tag",
  # and the system will look in the following locations to get the complete config:
       - /usr/lib/sota/conf.d/
       - /var/sota/sota.toml
       - /etc/sota/conf.d/
  fioctl config updates --tag -`,
	}
	cmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringP("group", "g", "", "Device group to use")
	configUpdatesCmd.Flags().StringP("tag", "", "", "Tag for devices to follow")
	configUpdatesCmd.Flags().StringP("tags", "", "", "Tag for devices to follow")
	configUpdatesCmd.Flags().StringP("apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolP("dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolP("force", "", false, "DANGER: For a config on a device that might result in corruption")
	_ = configUpdatesCmd.MarkFlagRequired("group")
	_ = configUpdatesCmd.Flags().MarkHidden("tags") // assign for go linter
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

	logrus.Debugf("Configuring group wide device updates for %s group %s", factory, group)
	opts.ListFunc = func() (*client.DeviceConfigList, error) {
		return api.GroupListConfig(factory, group)
	}
	opts.SetFunc = func(cfg client.ConfigCreateRequest, force bool) error {
		return api.GroupPatchConfig(factory, group, cfg, force)
	}
	subcommands.SetUpdatesConfig(&opts, "", nil)
}
