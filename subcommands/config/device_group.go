package config

import (
	"fmt"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	groupCmd := &cobra.Command{
		Use:   "device-group",
		Short: "Manage factory device groups",
	}
	cmd.AddCommand(groupCmd)

	groupCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Show available device groups",
		Run:   doListDeviceGroup,
	})
	groupCmd.AddCommand(&cobra.Command{
		Use:   "create <name> [<description>]",
		Short: "Create a new device groups",
		Run:   doCreateDeviceGroup,
		Args:  cobra.RangeArgs(1, 2),
	})
	groupCmd.AddCommand(&cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an existing device group",
		Run:   doDeleteDeviceGroup,
		Args:  cobra.ExactArgs(1),
	})

	updateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Rename an existing device group",
		Run:   doUpdateDeviceGroup,
		Args:  cobra.ExactArgs(1),
	}
	groupCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("name", "n", "", "Change a device group name")
	updateCmd.Flags().StringP("description", "d", "", "Change a device group description")
}

func doListDeviceGroup(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing a list of device groups for %s", factory)

	lst, err := api.FactoryListDeviceGroup(factory)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("NAME", "DESCRIPTION", "CREATED AT", "UPDATED AT")
	for _, grp := range *lst {
		t.AddLine(grp.Name, grp.Description, grp.ChangeMeta.CreatedAt, grp.ChangeMeta.UpdatedAt)
	}
	t.Print()
}

func doCreateDeviceGroup(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	logrus.Debugf("Creating a new device group %s for %s", name, factory)
	var description *string
	if len(args) > 1 {
		description = &args[1]
		logrus.Debugf("Description: %s", *description)
	}

	grp, err := api.FactoryCreateDeviceGroup(factory, name, description)
	subcommands.DieNotNil(err)

	fmt.Printf("Name: \t\t%s\n", grp.Name)
	if grp.Description != "" {
		fmt.Printf("Description: \t%s\n", grp.Description)
	}
	fmt.Printf("Created At: \t%s\n\n", grp.ChangeMeta.CreatedAt)
}

func doDeleteDeviceGroup(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	logrus.Debugf("Deleting a device group %s from %s", name, factory)

	err := api.FactoryDeleteDeviceGroup(factory, name)
	subcommands.DieNotNil(err)
}

func doUpdateDeviceGroup(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	old_name := args[0]
	var new_name, new_desc *string
	logrus.Debugf("Updating a device group %s for %s", old_name, factory)
	if cmd.Flags().Changed("name") {
		s, _ := cmd.Flags().GetString("name")
		logrus.Debugf("New name: %s", s)
		new_name = &s
	}
	if cmd.Flags().Changed("description") {
		s, _ := cmd.Flags().GetString("description")
		logrus.Debugf("New description: %s", s)
		new_desc = &s
	}

	if new_name == nil && new_desc == nil {
		subcommands.DieNotNil(fmt.Errorf("At least one attribute should be modified"))
	}

	err := api.FactoryPatchDeviceGroup(factory, old_name, new_name, new_desc)
	subcommands.DieNotNil(err)
}
