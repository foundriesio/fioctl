package devices

import (
	"fmt"
	"os"
	"strings"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "list-configured <device>",
		Short: "List remote actions configured on a device",
		Long:  "*NOTE*: Requires devices running LmP version 97 or later.",
		Run:   doListTriggers,
		Args:  cobra.ExactArgs(1),
	}
	triggersCmd.AddCommand(cmd)
}

func loadRemoteActions(d client.DeviceApi) []string {
	dcl, err := d.ListConfig()
	subcommands.DieNotNil(err)
	if len(dcl.Configs) > 0 {
		for _, cfgFile := range dcl.Configs[0].Files {
			if cfgFile.Name == "fio-remote-actions" {
				var actions []string
				for action := range strings.SplitSeq(cfgFile.Value, ",") {
					action = strings.TrimSpace(action)
					if len(action) > 0 {
						actions = append(actions, action)
					}
				}
				if actions == nil {
					break
				}
				return actions
			}
		}
	}
	return nil
}

func doListTriggers(cmd *cobra.Command, args []string) {
	name := args[0]

	// Quick sanity check for device
	d := getDevice(cmd, name)

	// See what triggers are allowed
	allowed := loadRemoteActions(d.Api)
	if len(allowed) == 0 {
		fmt.Println("Remote actions are not configured for this device")
		os.Exit(0)
	}

	fmt.Println("Available actions:")
	for _, trigger := range allowed {
		fmt.Println("*", trigger)
	}
}
