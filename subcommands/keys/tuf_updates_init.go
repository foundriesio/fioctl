package keys

import (
	"errors"
	"fmt"
	"os"

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
	initCmd.Flags().BoolP("first-time", "", false, "Used for the first customer rotation. The command will download the initial root key.")
	initCmd.Flags().StringP("keys", "k", "", "Path to <offline-creds.tgz> used to store initial root key.")
	_ = initCmd.MarkFlagFilename("keys")
	initCmd.MarkFlagsRequiredTogether("first-time", "keys")
	tufUpdatesCmd.AddCommand(initCmd)
}

func doTufUpdatesInit(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	changelog, _ := cmd.Flags().GetString("changelog")
	firstTime, _ := cmd.Flags().GetBool("first-time")
	keysFile, _ := cmd.Flags().GetString("keys")

	if firstTime {
		if _, err := os.Stat(keysFile); err == nil {
			subcommands.DieNotNil(errors.New(`Destination file exists.
Please make sure you aren't accidentally overwriting another Factory's keys`))
		}
	}

	res, err := api.TufRootUpdatesInit(factory, changelog, firstTime, isTufUpdatesShortcut)
	subcommands.DieNotNil(err)

	isTufUpdatesInitialized = true
	if !isTufUpdatesShortcut {
		fmt.Printf(`A new transaction to update TUF root keys started.
Your transaction ID is %s .
Please, keep it secret and only share with participants of the transaction.
Only the user who initiated the transaction can make changes to it without the transaction ID.
Other users are required to supply this transaction ID for all commands except review and cancel.
`,
			res.TransactionId)
	}

	if firstTime {
		creds := make(OfflineCreds)
		creds["tufrepo/keys/first-root.sec"] = []byte(res.FirstRootKeyPriv)
		creds["tufrepo/keys/first-root.pub"] = []byte(res.FirstRootKeyPub)
		saveTufCreds(keysFile, creds)
	}
}
