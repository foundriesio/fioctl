package keys

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	del := &cobra.Command{
		Use:   "delete-offline-key --role root|targets --txid=<txid> --keys=<tuf-root-keys.tgz>|--key-id=<key-id>",
		Short: "Stage deletion of the offline TUF signing key for the Factory",
		Long: `Stage deletion of the offline TUF signing key for the Factory.

There are two ways to delete the offline TUF signing key:
- If you own the keys file - you can delete your key by providing your keys file.
  Fioctl will search through your keys file for an appropriate key to delete.
- You can also provide an exact key ID to delete.

When you delete the TUF targets offline signing key:
- if there are production targets in your factory, corresponding signatures are also deleted.
  if any production targets lack enough signatures - you need to sign them using the "sign-prod-targets" command.
- if there is an active wave in your factory, the TUF targets key deletion is not allowed.`,
		Example: `
- Delete offline TUF root key:
  fioctl keys tuf updates delete-offline-key \
    --txid=abc --role=root --keys=tuf-root-keys.tgz
- Delete offline TUF targets key by its ID:
  fioctl keys tuf updates delete-offline-key \
    --txid=abc --role=targets
	 --key-id=15bbb6e79c9ac73b2db7df73c96f3a4937a25d948c048ba0208e49e426e5888a`,
		Run: doTufUpdatesDeleteOfflineKey,
	}
	del.Flags().StringP("role", "r", "", "TUF role name, supported: Root, Targets.")
	_ = del.MarkFlagRequired("role")
	del.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	del.Flags().StringP("keys", "k", "", "Path to <tuf-root-keys.tgz> used to sign TUF root.")
	_ = del.MarkFlagFilename("keys")
	del.Flags().StringP("key-id", "i", "", "A key ID to delete, as specified in your TUF root.")
	del.MarkFlagsMutuallyExclusive("keys", "key-id")
	tufUpdatesCmd.AddCommand(del)
}

func doTufUpdatesDeleteOfflineKey(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")
	keysFile, _ := cmd.Flags().GetString("keys")
	keyId, _ := cmd.Flags().GetString("key-id")
	roleName, _ := cmd.Flags().GetString("role")
	roleName = ParseTufRoleNameOffline(roleName)

	var (
		creds OfflineCreds
		err   error
	)

	if keysFile != "" {
		creds, err = GetOfflineCreds(keysFile)
		subcommands.DieNotNil(err)
	} else if keyId == "" {
		subcommands.DieNotNil(errors.New(
			"Either --keys or --key-id option is required to delete the offline TUF key.",
		))
	}

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)
	_, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)

	var (
		roleToUpdate *tuf.RootRole
		validKeyIds  []string
	)

	switch roleName {
	case tufRoleNameRoot:
		roleToUpdate = newCiRoot.Signed.Roles["root"]
		validKeyIds = roleToUpdate.KeyIDs
	case tufRoleNameTargets:
		roleToUpdate = newCiRoot.Signed.Roles["targets"]
		onlineTargetsId := updates.Updated.OnlineKeys["targets"]
		if onlineTargetsId == "" {
			subcommands.DieNotNil(errors.New("Unable to find online target key for factory"))
		}
		if keyId != "" && keyId == onlineTargetsId {
			subcommands.DieNotNil(fmt.Errorf(
				"It is not allowed to delete an online TUF targets key: %s", onlineTargetsId,
			))
		}
		validKeyIds = subcommands.SliceRemove(roleToUpdate.KeyIDs, onlineTargetsId)
	default:
		panic(fmt.Errorf("Unexpected role name: %s", roleName))
	}

	fmt.Println("= Delete keyid:", keyId)
	if keyId == "" {
		oldKey, err := FindOneTufSigner(newCiRoot, creds, validKeyIds)
		subcommands.DieNotNil(err)
		keyId = oldKey.Id
	} else if !slices.Contains(validKeyIds, keyId) {
		subcommands.DieNotNil(fmt.Errorf(
			"Key ID %s not found in TUF %s role keys: %v", keyId, roleName, validKeyIds,
		))
	}
	roleToUpdate.KeyIDs = subcommands.SliceRemove(roleToUpdate.KeyIDs, keyId)
	newCiRoot, newProdRoot = finalizeTufRootChanges(newCiRoot, newProdRoot)

	fmt.Println("= Uploading new TUF root")
	subcommands.DieNotNil(api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil))
}
