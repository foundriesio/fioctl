package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	deleteCmd := &cobra.Command{
		Use:   "delete <file>",
		Short: "Delete file from the current configuration",
		Run:   doConfigDelete,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("group", "g", "", "Device group to use")
}

func doConfigDelete(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	group, _ := cmd.Flags().GetString("group")
	filename := args[0]

	if group == "" {
		logrus.Debugf("Deleting file %s from config for %s", filename, factory)
		subcommands.DieNotNil(api.FactoryDeleteConfig(factory, filename))
	} else {
		logrus.Debugf("Deleting file %s from config for %s group %s", filename, factory, group)
		subcommands.DieNotNil(api.GroupDeleteConfig(factory, group, filename))
	}
}
