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
currently configured. Configuration can be updated with commands
like:

  # Make devices start taking updates from Targets tagged with "devel"
  fioctl config updates --tags devel

  # Make devices start taking updates from 2 different tags:
  fioctl config updates --tags devel,devel-foo

  # Set the docker apps that devices will run:
  fioctl config updates --apps shellhttpd

  # Set the docker apps and the tag for devices:
  fioctl config updates --apps shellhttpd --tags master

  # Migrate devices from old docker-apps to compose-apps:
  fioctl config updates --compose-apps

Use a -g or --group parameter to view or change a device group wide configuration instead.
`,
	}
	cmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringP("group", "g", "", "Device group to use")
	configUpdatesCmd.Flags().StringP("tags", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().StringP("apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolP("compose-apps", "", false, "Migrate device from docker-apps to compose-apps")
	configUpdatesCmd.Flags().StringP("compose-dir", "", "", "The directory to install compose apps in")
	configUpdatesCmd.Flags().BoolP("dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolP("force", "", false, "DANGER: For a config on a device that might result in corruption")

	_ = configUpdatesCmd.Flags().MarkHidden("compose-dir") // assign for go linter
}

func doConfigUpdates(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	group, _ := cmd.Flags().GetString("group")
	updateApps, _ := cmd.Flags().GetString("apps")
	updateTags, _ := cmd.Flags().GetString("tags")
	setComposeApps, _ := cmd.Flags().GetBool("compose-apps")
	composeAppsDir, _ := cmd.Flags().GetString("compose-dir")
	isDryRun, _ := cmd.Flags().GetBool("dryrun")
	isForced, _ := cmd.Flags().GetBool("force")

	opts := subcommands.SetUpdatesConfigOptions{
		UpdateApps:     updateApps,
		UpdateTags:     updateTags,
		SetComposeApps: setComposeApps,
		ComposeAppsDir: composeAppsDir,
		IsDryRun:       isDryRun,
		IsForced:       isForced,
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
	subcommands.SetUpdatesConfig(&opts)
}
