package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Show a changelog of configuration",
		Run:   doConfigLog,
	}
	cmd.AddCommand(logCmd)
	logCmd.Flags().StringP("group", "g", "", "Device group to use")
	logCmd.Flags().IntP("limit", "n", 0, "Limit the number of results displayed")
}

func doConfigLog(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	listLimit, _ := cmd.Flags().GetInt("limit")
	group, _ := cmd.Flags().GetString("group")

	if group == "" {
		logrus.Debugf("Showing config history for %s", factory)
		subcommands.LogConfigs(&subcommands.LogConfigsOptions{
			Limit: listLimit,
			ListFunc: func() (*client.DeviceConfigList, error) {
				return api.FactoryListConfig(factory)
			},
			ListContFunc: api.FactoryListConfigCont,
		})
	} else {
		logrus.Debugf("Showing config history for %s group %s", factory, group)
		subcommands.LogConfigs(&subcommands.LogConfigsOptions{
			Limit: listLimit,
			ListFunc: func() (*client.DeviceConfigList, error) {
				return api.GroupListConfig(factory, group)
			},
			ListContFunc: api.GroupListConfigCont,
		})
	}
}
