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
When run with no options, this command will print out how the device is
currently configured and reporting.`,
		Example: `
  # Make a device start taking updates from Targets tagged with "devel"
  fioctl devices config updates <device> --tags devel

  # Make a device start taking updates from 2 different tags:
  fioctl devices config updates <device> --tags devel,devel-foo

  # Set the docker apps a device will run:
  fioctl devices config updates <device> --apps shellhttpd

  # Set the docker apps and the tag:
  fioctl devices config updates <device> --apps shellhttpd --tags master

  # Move device from old docker-apps to compose-apps:
  fioctl devices config updates <device> --compose-apps
`,
	}
	configCmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringP("tags", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().StringP("apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolP("compose-apps", "", false, "Migrate device from docker-apps to compose-apps")
	configUpdatesCmd.Flags().StringP("compose-dir", "", "", "The directory to install compose apps in")
	configUpdatesCmd.Flags().BoolP("dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolP("force", "", false, "DANGER: For a config on a device that might result in corruption")

	_ = configUpdatesCmd.Flags().MarkHidden("compose-dir") // assign for go linter
}

func doConfigUpdates(cmd *cobra.Command, args []string) {
	name := args[0]
	updateApps, _ := cmd.Flags().GetString("apps")
	updateTags, _ := cmd.Flags().GetString("tags")
	setComposeApps, _ := cmd.Flags().GetBool("compose-apps")
	composeAppsDir, _ := cmd.Flags().GetString("compose-dir")
	isDryRun, _ := cmd.Flags().GetBool("dryrun")
	isForced, _ := cmd.Flags().GetBool("force")

	logrus.Debugf("Configuring device updates for %s", name)

	device, err := api.DeviceGet(name)
	subcommands.DieNotNil(err, "Failed to fetch a device:")

	subcommands.SetUpdatesConfig(&subcommands.SetUpdatesConfigOptions{
		UpdateApps:     updateApps,
		UpdateTags:     updateTags,
		SetComposeApps: setComposeApps,
		ComposeAppsDir: composeAppsDir,
		IsDryRun:       isDryRun,
		IsForced:       isForced,
		Device:         device,
		ListFunc: func() (*client.DeviceConfigList, error) {
			return api.DeviceListConfig(name)
		},
		SetFunc: func(cfg client.ConfigCreateRequest, force bool) error {
			return api.DevicePatchConfig(name, cfg, force)
		},
	})
}
