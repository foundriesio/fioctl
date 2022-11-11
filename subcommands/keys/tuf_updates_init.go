package keys

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Start a new transaction to update TUF root keys",
		Run:   doTufUpdatesInit,
	}
	initCmd.Flags().StringP("changelog", "m", "", "Reason for doing this operation. Saved in root metadata to track change history.")
	_ = initCmd.MarkFlagRequired("changelog")
	tufUpdatesCmd.AddCommand(initCmd)
}

func doTufUpdatesInit(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	changelog, _ := cmd.Flags().GetString("changelog")

	res, err := api.TufRootUpdatesInit(factory, changelog)
	subcommands.DieNotNil(err)

	fmt.Printf(`A new transaction to update TUF root keys started.
Your transaction ID is %s .
Please, keep it secret and only share with participants of the transaction.
Nobody can make changes to the transaction without this ID other than cancel it.
`,
		res.TransactionId)
}
