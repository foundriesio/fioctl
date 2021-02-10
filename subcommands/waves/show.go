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
		Use:   "show <wave>",
		Short: "Show a given wave by name",
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
		data, _ := json.MarshalIndent(wave.Targets, "  ", "  ")
		fmt.Println("  " + string(data))
	}
}
