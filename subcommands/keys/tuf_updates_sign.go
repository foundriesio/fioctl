package keys

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	signCmd := &cobra.Command{
		Use:   "sign --txid=<txid> --keys=<tuf-root-keys.tgz>",
		Short: "Sign the staged TUF root for your Factory with the offline root key",
		Long:  "Sign the staged TUF root for your Factory with the offline root key",
		Run:   doTufUpdatesSign,
	}
	signCmd.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	signCmd.Flags().StringP("keys", "k", "", "Path to <tuf-root-keys.tgz> used to sign TUF root.")
	_ = signCmd.MarkFlagFilename("keys")
	_ = signCmd.MarkFlagRequired("keys")
	tufUpdatesCmd.AddCommand(signCmd)
}

func doTufUpdatesSign(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")
	keysFile, _ := cmd.Flags().GetString("keys")

	creds, err := GetOfflineCreds(keysFile)
	subcommands.DieNotNil(err)

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	curCiRoot, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)
	if newProdRoot == nil {
		// User might still want to re-sign and apply updates even if there are no changes.
		// E.g. this way the user can optimize the latest root.json size after the root key rotation
		newProdRoot = genProdTufRoot(newCiRoot)
	}
	signNewTufRoot(curCiRoot, newCiRoot, newProdRoot, creds)

	fmt.Println("= Uploading new TUF root")
	subcommands.DieNotNil(api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil, nil))
}
