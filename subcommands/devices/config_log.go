package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	logConfigCmd := &cobra.Command{
		Use:   "log <device>",
		Short: "Show a changelog of device's configuration",
		Run:   doConfigLog,
		Args:  cobra.ExactArgs(1),
	}
	configCmd.AddCommand(logConfigCmd)
	logConfigCmd.Flags().IntP("limit", "n", 0, "Limit the number of results displayed.")
}

func doConfigLog(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	device := args[0]
	listLimit, _ := cmd.Flags().GetInt("limit")
	logrus.Debugf("Showing device config log for %s", device)

	lookups, err := api.UsersGetLookups(factory)
	subcommands.DieNotNil(err)

	subcommands.LogConfigs(&subcommands.LogConfigsOptions{
		Limit:         listLimit,
		ShowAppliedAt: true,
		ListFunc: func() (*client.DeviceConfigList, error) {
			return api.DeviceListConfig(factory, device)
		},
		ListContFunc: api.DeviceListConfigCont,
		UserLookup:   lookups,
	})
}
