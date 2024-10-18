package secrets

import (
	"fmt"
	"os"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List secret credentials configured in the Factory",
		Run:   doList,
	})
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing Factory secrets for: %s", factory)

	triggers, err := api.FactoryTriggers(factory)
	subcommands.DieNotNil(err)

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
