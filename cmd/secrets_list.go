package cmd

import (
	"fmt"
	"os"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secret credentials configured in the factory",
	Run:   doSecretList,
}

func init() {
	secretsCmd.AddCommand(secretsListCmd)
}

func doSecretList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing factory secrets for: %s", factory)

	triggers, err := api.FactoryTriggers(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	t := tabby.New()
	t.AddHeader("SECRETS")
	if len(triggers) == 1 {
		for _, secret := range triggers[0].Secrets {
			t.AddLine(secret.Name)
		}
	} else if len(triggers) != 0 {
		fmt.Println("ERROR: Factory configuration issue. Factory has unexpected number of triggers.")
		os.Exit(1)
	}
	t.Print()
}
