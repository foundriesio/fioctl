package devices

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	logConfigCmd := &cobra.Command{
		Use:   "log <device>",
		Short: "Show a change log of device's configuration",
		Run:   doConfigLog,
		Args:  cobra.ExactArgs(1),
	}
	configCmd.AddCommand(logConfigCmd)
	logConfigCmd.Flags().IntP("limit", "n", 0, "Limit the number of results displayed.")
}

func doConfigLog(cmd *cobra.Command, args []string) {
	device := args[0]
	listLimit, _ := cmd.Flags().GetInt("limit")
	logrus.Debugf("Showing device config log for %s", device)

	subcommands.LogConfigs(&subcommands.LogConfigsOptions{
		Limit:         listLimit,
		ShowAppliedAt: true,
		ListFunc: func() (*client.DeviceConfigList, error) {
			return api.DeviceListConfig(device)
		},
		ListContFunc: api.DeviceListConfigCont,
	})
}
