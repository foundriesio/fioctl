package ci

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	secretsCmd.AddCommand(&cobra.Command{
		Use:   "update secret_name=secret_val...",
		Short: "Update secret(s) in a factory",
		Example: `
  # Create or update a secret
  fioctl ci secrets update githubtok=foo

  # Create or update a secret with value from a file
  fioctl ci secrets update ssh-github.key==/tmp/ssh-github.key

  # Delete a secret by setting it to an empty value. eg:
  fioctl ci secrets update secret_name=`,
		Run:  doUpdate,
		Args: cobra.MinimumNArgs(1),
	})
}

func doUpdate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Updating factory secrets for: %s", factory)

	triggers, err := api.FactoryTriggers(factory)
	subcommands.DieNotNil(err)

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
		} else if value[0] == '=' {
			bytes, err := os.ReadFile(value[1:])
			subcommands.DieNotNil(err, "Unable to read secret:")
			content := string(bytes)
			secrets[i].Value = &content
		} else {
			secrets[i].Value = &value
		}
	}
	logrus.Debugf("Secrets are: %v", secrets)

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
	subcommands.DieNotNil(api.FactoryUpdateTrigger(factory, pt))
}
