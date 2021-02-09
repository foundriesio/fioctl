package waves

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show Ra given wave",
		Run:   doShowWave,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(showCmd)
	showCmd.Flags().BoolP("show-targets", "s", false, "Show wave targets")
}

func doShowWave(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	name := args[0]
	showTargets, _ := cmd.Flags().GetBool("show-targets")
	logrus.Debugf("Showing a wave %s for %s", name, factory)

	wave, err := api.FactoryGetWave(factory, name, showTargets)
	subcommands.DieNotNil(err)

	fmt.Printf("Name: \t\t%s\n", wave.Name)
	fmt.Printf("Version: \t%s\n", wave.Version)
	fmt.Printf("Tag: \t\t%s\n", wave.Tag)
	fmt.Printf("Status: \t%s\n", wave.Status)
	fmt.Printf("Created At: \t%s\n", wave.CreatedAt)
	if wave.FinishedAt != "" {
		fmt.Printf("Finished At: \t\t%s\n", wave.FinishedAt)
	}
	if showTargets {
		fmt.Printf("Targets:\n")
		// No need to canonicalize, a user may call an API like below to view raw JSON from database:
		// fioctl get /ota/factories/<factory>/waves/<wave>/?raw-targets=1 | jq '.targets|fromjson'
		// I'd like to show YAML here, but it shows TUF []byte signatures as lists.
		meta, _ := json.MarshalIndent(wave.Targets, "  ", "  ")
		fmt.Println("  " + string(meta))
	}
}
