package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	configUpdatesCmd := &cobra.Command{
		Use:   "updates <device>",
		Short: "Configure aktualizr-lite settings for how updates are applied to a device",
		Run:   doConfigUpdates,
		Args:  cobra.ExactArgs(1),
		Long: `View or change configuration parameters used by aktualizr-lite for updating a device.
When run with no options, this command print out how the device is
currently configured and reporting.`,
		Example: `
  # Make a device start taking updates from Targets tagged with "devel"
  fioctl devices config updates <device> --tag devel

  # Set the Compose apps a device will run:
  fioctl devices config updates <device> --apps shellhttpd

  # Set the Compose apps and the tag:
  fioctl devices config updates <device> --apps shellhttpd --tag master

  # There are two special characters: "," and "-".
  # Providing a "," sets the Compose apps to "none", meaning it will run no apps:
  fioctl devices config updates <device> --apps ,

  # Providing a "-" sets the Compose apps to "preset-apps" (all apps on most devices).
  # The system looks in the following locations to get the complete config:
       - /usr/lib/sota/conf.d/
       - /var/sota/sota.toml
       - /etc/sota/conf.d/
  fioctl devices config updates <device> --apps -
  
  # Set the device tag to a preset-tag,
  # and the system will look in the following locations in order to get the complete config:
       - /usr/lib/sota/conf.d/
       - /var/sota/sota.toml
       - /etc/sota/conf.d/
  fioctl devices config updates <device> --tag -`,
	}
	configCmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringP("tag", "", "", "Target tag for device to follow")
	configUpdatesCmd.Flags().StringP("tags", "", "", "Target tag for device to follow")
	configUpdatesCmd.Flags().StringP("apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolP("dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolP("force", "", false, "DANGER: For a config on a device that might result in corruption")

	_ = configUpdatesCmd.Flags().MarkHidden("tags") // assign for go linter
}

func doConfigUpdates(cmd *cobra.Command, args []string) {
	name := args[0]
	updateApps, _ := cmd.Flags().GetString("apps")
	updateTag, _ := cmd.Flags().GetString("tag")
	if len(updateTag) == 0 {
		// check the old, deprecated "tags" option
		updateTag, _ = cmd.Flags().GetString("tags")
	}
	isDryRun, _ := cmd.Flags().GetBool("dryrun")
	isForced, _ := cmd.Flags().GetBool("force")

	logrus.Debugf("Configuring device updates for %s", name)

	device := getDevice(cmd, name)

	subcommands.SetUpdatesConfig(&subcommands.SetUpdatesConfigOptions{
		UpdateApps: updateApps,
		UpdateTag:  updateTag,
		IsDryRun:   isDryRun,
		IsForced:   isForced,
		Device:     device,
		ListFunc: func() (*client.DeviceConfigList, error) {
			return device.Api.ListConfig()
		},
		SetFunc: func(cfg client.ConfigCreateRequest, force bool) error {
			return device.Api.PatchConfig(cfg, force)
		},
	},
		device.Tag, device.DockerApps)
}
