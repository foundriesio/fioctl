package devices

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	device := args[0]
	unset, _ := cmd.Flags().GetBool("unset")
	var group string
	if unset {
		if len(args) == 2 {
			fmt.Println("ERROR: Cannot assign and unset a device group in one command")
			os.Exit(1)
		}
		group = ""
		logrus.Debugf("Unsetting a device group from device %s", device)
	} else {
		if len(args) == 1 {
			fmt.Println("ERROR: Either device group or --unset option must be provided")
			os.Exit(1)
		}
		group = args[1]
		logrus.Debugf("Assigning device %s to group %d", device, group)
	}

	err := api.DeviceSetGroup(device, group)
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}
}
