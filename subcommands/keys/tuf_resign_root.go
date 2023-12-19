package keys

import (
	"fmt"
	"time"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	resign := &cobra.Command{
		Use:    "resign-root <offline key archive>",
		Short:  "Re-sign the Factory's TUF root metadata",
		Hidden: true,
		Run:    doResignRoot,
		Args:   cobra.ExactArgs(1),
		Deprecated: `it will be removed in the future.
Please, use a more secure way to keep your TUF root role fresh:
- rotate the root key using "fioctl keys tuf rotate-offline-key --role=root --keys=<offline-creds.tgz>".
`,
	}
	resign.Flags().StringP("changelog", "m", "Refresh TUF root expiration by 1 year",
		"Reason for re-signing root.json. Saved in root metadata to track change history.")
	cmd.AddCommand(resign)
}

func doResignRoot(cmd *cobra.Command, args []string) {
	isTufUpdatesShortcut = true
	factory := viper.GetString("factory")
	changelog, _ := cmd.Flags().GetString("changelog")
	keysFile := args[0]

	creds, err := GetOfflineCreds(keysFile)
	subcommands.DieNotNil(err)

	// Below the `tuf updates` subcommands are chained in a correct order.
	// Detach from the parent, so that command calls below use correct args.
	tufCmd.RemoveCommand(tufUpdatesCmd)

	fmt.Println("= Creating new TUF updates transaction")
	tufUpdatesCmd.SetArgs([]string{"init", "-m", changelog})
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	fmt.Println("= Extending TUF root expiration")
	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	curCiRoot, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)
	newCiRoot.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	newCiRoot, newProdRoot = finalizeTufRootChanges(newCiRoot, newProdRoot)
	signNewTufRoot(curCiRoot, newCiRoot, newProdRoot, creds)

	fmt.Println("= Uploading new TUF root")
	subcommands.DieNotNil(api.TufRootUpdatesPut(factory, "", newCiRoot, newProdRoot, nil, nil))

	fmt.Println("= Applying staged TUF root changes")
	tufUpdatesCmd.SetArgs([]string{"apply"})
	subcommands.DieNotNil(tufUpdatesCmd.Execute())
}
