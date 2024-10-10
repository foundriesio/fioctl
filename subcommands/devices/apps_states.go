package devices

import (
	"fmt"
	"strings"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

var (
	asListLimit int
)

func init() {
	appsStatesCmd := &cobra.Command{
		Use:   "apps-states <name>",
		Short: "List the states of Apps reported by a device",
		Run:   doListStates,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(appsStatesCmd)
	appsStatesCmd.Flags().IntVarP(&asListLimit, "limit", "n", 1, "Limit the number of App states to display.")
}

func doListStates(cmd *cobra.Command, args []string) {
	d := getDeviceApi(cmd, args[0])
	states, err := d.GetAppsStates()
	subcommands.DieNotNil(err)

	printAppsState := func(appsState map[string]client.AppState, stateFilter string, filterIn bool) {
		for name, state := range appsState {
			if len(stateFilter) > 0 {
				if filterIn && state.State != stateFilter {
					continue
				}
				if !filterIn && state.State == stateFilter {
					continue
				}
			}
			if len(state.Uri) > 0 {
				fmt.Printf("\t%s\t%s\n", name, state.Uri)
			} else {
				fmt.Printf("\t%s\n", name)
			}
			for _, srv := range state.Services {
				fmt.Printf("\t\t%s:\n", srv.Name)
				fmt.Printf("\t\t\tURI:\t%s\n", srv.ImageUri)
				fmt.Printf("\t\t\tHash:\t%s\n", srv.Hash)
				fmt.Printf("\t\t\tHealth:\t%s\n", srv.Health)
				fmt.Printf("\t\t\tState:\t%s\n", srv.State)
				fmt.Printf("\t\t\tStatus:\t%s\n", srv.Status)
				if len(srv.Logs) > 0 {
					fmt.Println("\t\t\tLogs:")
					for _, line := range strings.Split(srv.Logs, "\n") {
						fmt.Printf("\t\t\t | %s\n", line)
					}
				}
			}
			fmt.Println()
		}
	}
	for indx, s := range states.States {
		if indx >= asListLimit {
			break
		}
		fmt.Printf("Time:\t%s\n", s.DeviceTime)
		fmt.Printf("Hash:\t%s\n", s.Ostree)
		fmt.Println("Unhealthy Apps:")
		printAppsState(s.Apps, "healthy", false)
		fmt.Println("Healthy Apps:")
		printAppsState(s.Apps, "healthy", true)
		fmt.Println()
	}
}
