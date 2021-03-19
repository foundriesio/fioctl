package status

import (
	"fmt"

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
	subcommands.DieNotNil(err)

	fmt.Println("Total number of devices:", status.TotalDevices)

	if len(status.ProdTags) > 0 || len(status.ProdWaveTags) > 0 {
		fmt.Println("\nProduction devices:")
		t := tabby.New()
		t.AddHeader("TAG", "LATEST TARGET", "DEVICES", "ON LATEST", "ON ORPHAN", "ONLINE")
		for idx, tag := range append(status.ProdWaveTags, status.ProdTags...) {
			name := tag.Name
			if idx < len(status.ProdWaveTags) {
				// Wave devices are always tagged
				name += " (wave)"
			} else if len(name) == 0 {
				// Prod devices can be untagged, although they cannot fetch updates
				name = "(Untagged)"
			}
			t.AddLine(name, tag.LatestTarget, tag.DevicesTotal, tag.DevicesOnLatest,
				tag.DevicesOnOrphan, tag.DevicesOnline)
		}
		t.Print()

		fmt.Println("\nTest devices:")
	}

	t := tabby.New()
	t.AddHeader("TAG", "LATEST TARGET", "DEVICES", "ON LATEST", "ONLINE")
	for _, tag := range status.Tags {
		name := tag.Name
		if len(name) == 0 {
			// Test devices can be untagged and can even fetch updates
			name = "(Untagged)"
		}
		t.AddLine(name, tag.LatestTarget, tag.DevicesTotal, tag.DevicesOnLatest, tag.DevicesOnline)
	}
	t.Print()

	fmt.Println("\nOrphan target versions below are marked with a star (*)")
	printTargetStatus("Active Wave", status.ProdWaveTags)
	printTargetStatus("Production", status.ProdTags)
	printTargetStatus("CI", status.Tags)
}

func printTargetStatus(tagPrefix string, tagStatus []client.TagStatus) {
	for _, tag := range tagStatus {
		name := tag.Name
		if len(name) == 0 {
			// These are untagged devices:
			name = "(Untagged)"
		}
		fmt.Printf("\n## %s Tag: %s\n", tagPrefix, name)
		// Tabby doesn't indent (or at least easily) so:
		fmt.Println("\tTARGET   DEVICES  INSTALLING  DETAILS")
		fmt.Println("\t-------  -------  ----------  -------")
		for _, tgt := range tag.Targets {
			var orphan, details string
			if tgt.IsOrphan {
				orphan = "*"
			}
			if tgt.Version > 0 {
				details = fmt.Sprintf("`fioctl targets show %d`", tgt.Version)
			}
			fmt.Printf("\t%-6d%-1s  %-7d  %-10d  %s\n",
				tgt.Version, orphan, tgt.Devices, tgt.Reinstalling, details)
		}
	}
}
