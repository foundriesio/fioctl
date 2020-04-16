package status

import (
	"fmt"
	"os"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var api *client.Api

var inactiveThreshold int

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get dashboard view of a factory and its devices",
		Run:   showStatus,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = subcommands.Login(cmd)
		},
	}
	subcommands.RequireFactory(cmd)
	cmd.Flags().IntVarP(&inactiveThreshold, "offline-threshold", "", 4, "Consider device 'OFFLINE' if not seen in the last X hours")
	return cmd
}

func showStatus(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing status of %s", factory)

	status, err := api.FactoryStatus(factory, inactiveThreshold)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}

	fmt.Println("Total number of devices:", status.TotalDevices)
	fmt.Println("")
	t := tabby.New()
	t.AddHeader("TAG", "LATEST TARGET", "DEVICES", "ON LATEST", "ONLINE")
	for _, tag := range status.Tags {
		if len(tag.Name) == 0 {
			// These are untagged devices:
			tag.Name = "(Untagged)"
		}
		t.AddLine(tag.Name, tag.LatestTarget, tag.DevicesTotal, tag.DevicesOnLatest, tag.DevicesOnline)
	}
	t.Print()

	for _, tag := range status.Tags {
		if len(tag.Name) == 0 {
			// These are untagged devices:
			tag.Name = "(Untagged)"
		}
		fmt.Println("\n## Tag:", tag.Name)
		// Tabby doesn't indent (or at least easily) so:
		fmt.Println("\tTARGET  DEVICES  DETAILS")
		fmt.Println("\t------  -------  -------")
		for _, tgt := range tag.Targets {
			fmt.Printf("\t%-6d  %-7d  `fioctl targets show %d`\n", tgt.Version, tgt.Devices, tgt.Version)
		}
	}
}
