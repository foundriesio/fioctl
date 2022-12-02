package keys

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply staged TUF root updates for the Factory",
		Run:   doTufUpdatesApply,
	}
	applyCmd.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	_ = applyCmd.MarkFlagRequired("txid")
	tufUpdatesCmd.AddCommand(applyCmd)
}

func doTufUpdatesApply(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")

	err := api.TufRootUpdatesApply(factory, txid)
	if err != nil {
		msg := "Failed to apply staged TUF root updates:\n%w\n"
		var isNonFatal bool
		if herr := client.AsHttpError(err); herr != nil {
			isNonFatal = slices.Contains([]int{400, 401, 403, 422, 423}, herr.Response.StatusCode)
		}
		if isNonFatal {
			msg += `No changes were made to your Factory.
There are two options available for you now:
- fix the errors listed above and run the "fioctl keys tuf updates apply" again.
- cancel the staged TUF root updates using the "fioctl keys updates cancel"`
		} else {
			msg += `
This is a critical error: Staged TUF root updates may be partially applied to your Factory.
Please, re-run the "fioctl keys tuf updates apply" again after some time, but as soon as possible.
If the error persists, please, contact the customer support.`
		}
		err = fmt.Errorf(msg, err)
	}
	subcommands.DieNotNil(err)

	fmt.Println(`The staged TUF root updates were applied to your Factory.
Please, make sure that the updated TUF keys file(s) are stored in a safe place.`)
}
