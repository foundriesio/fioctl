package keys

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel staged TUF root updates for the Factory",
		Run:   doTufUpdatesCancel,
	}
	tufUpdatesCmd.AddCommand(cancelCmd)
}

func doTufUpdatesCancel(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	subcommands.DieNotNil(api.TufRootUpdatesCancel(factory))
	fmt.Println(`The staged TUF root updates were canceled.
No other changes were made to your Factory.`)
}
