package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

var secretsUpdateCmd = &cobra.Command{
	Use:   "update secret_name=secret_val...",
	Short: "Update secret(s) in a factory",
	Run:   doSecretsUpdate,
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	secretsCmd.AddCommand(secretsUpdateCmd)
}

func doSecretsUpdate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Updating factory secrets for: %s", factory)

	triggers, err := api.FactoryTriggers(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	secrets := make([]client.ProjectSecret, len(args))
	for i, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			fmt.Println("ERROR: Invalid key=value argument: ", arg)
			os.Exit(1)
		}
		secrets[i].Name = parts[0]
		value := parts[1]
		if value == "" {
			secrets[i].Value = nil
		} else {
			secrets[i].Value = &value
		}
	}
	logrus.Debugf("Secrets are: %s", secrets)

	var pt client.ProjectTrigger

	if len(triggers) == 0 {
		pt = client.ProjectTrigger{Type: "simple"}
	} else if len(triggers) == 1 {
		pt = triggers[0]
	} else {
		fmt.Println("ERROR: Factory configuration issue. Factory has unexpected number of triggers.")
		os.Exit(1)
	}

	pt.Secrets = secrets
	if err := api.FactoryUpdateTrigger(factory, pt); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
