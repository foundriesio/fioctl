package devices

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	groupCmd := &cobra.Command{
		Use:   "group <device> [<group>]",
		Short: "Assign a device to an existing factory device group",
		Run:   doConfigGroup,
		Args:  cobra.RangeArgs(1, 2),
	}
	groupCmd.Flags().Bool("unset", false, "Unset an associated device group")
	configCmd.AddCommand(groupCmd)
}

func doConfigGroup(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	device := args[0]
	unset, _ := cmd.Flags().GetBool("unset")
	var group string
	if unset {
		if len(args) == 2 {
			subcommands.DieNotNil(fmt.Errorf("Cannot assign and unset a device group in one command"))
		}
		group = ""
		logrus.Debugf("Unsetting a device group from device %s", device)
	} else {
		if len(args) == 1 {
			subcommands.DieNotNil(fmt.Errorf("Either device group or --unset option must be provided"))
		}
		group = args[1]
		logrus.Debugf("Assigning device %s to group %s", device, group)
	}

	err := api.DeviceSetGroup(factory, device, group)
	subcommands.DieNotNil(err)
}
