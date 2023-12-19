package keys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	add := &cobra.Command{
		Use:   "add-offline-key --role root|targets --txid=<txid> --keys=<tuf-root-keys.tgz>",
		Short: "Stage addition of the offline TUF signing key for the Factory",
		Long: `Stage addition of the offline TUF signing key for the Factory.

The new offline signing key will be used in both CI and production TUF root.

When you add a new TUF targets offline signing key, existing production targets are not signed by it.
Please, use the sign-prod-targets subcommand if you want to sign existing production targets with a new key.`,
		Example: `
- Add offline TUF root key:
  fioctl keys tuf updates add-offline-key \
	 --txid=abc --role=root --keys=tuf-root-keys.tgz
- Add offline TUF targets key, explicitly specifying new key type (and signing algorithm):
  fioctl keys tuf updates add-offline-key \
    --txid=abc --role=targets --keys=tuf-targets-keys.tgz --key-type=ed25519`,
		Run: doTufUpdatesAddOfflineKey,
	}
	add.Flags().StringP("role", "r", "", "TUF role name, supported: Root, Targets.")
	_ = add.MarkFlagRequired("role")
	add.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	add.Flags().StringP("keys", "k", "", `Path to <tuf-keys.tgz> where a new key should be created.
For security reasons, it is disallowed to add several actual keys for the same TUF role into the same file.`)
	_ = add.MarkFlagFilename("keys")
	_ = add.MarkFlagRequired("keys")
	add.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519, "Key type, supported: Ed25519, RSA.")
	tufUpdatesCmd.AddCommand(add)
}

func doTufUpdatesAddOfflineKey(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")
	keyTypeStr, _ := cmd.Flags().GetString("key-type")
	keyType := ParseTufKeyType(keyTypeStr)
	keysFile, _ := cmd.Flags().GetString("keys")
	roleName, _ := cmd.Flags().GetString("role")
	roleName = ParseTufRoleNameOffline(roleName)

	var creds OfflineCreds
	if _, err := os.Stat(keysFile); err == nil {
		creds, err = GetOfflineCreds(keysFile)
		subcommands.DieNotNil(err)
		subcommands.AssertWritable(keysFile)
	} else if errors.Is(err, fs.ErrNotExist) {
		creds = make(OfflineCreds, 0)
		saveTufCreds(keysFile, creds)
	} else {
		subcommands.DieNotNil(err)
	}

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	_, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)

	switch roleName {
	case tufRoleNameRoot:
		addOfflineRootKey(newCiRoot, creds, keyType)
	case tufRoleNameTargets:
		onlineTargetsId := updates.Updated.OnlineKeys["targets"]
		if onlineTargetsId == "" {
			subcommands.DieNotNil(errors.New("Unable to find online target key for factory"))
		}
		addOfflineTargetsKey(newCiRoot, creds, keyType, onlineTargetsId)
	default:
		panic(fmt.Errorf("Unexpected role name: %s", roleName))
	}
	newCiRoot, newProdRoot = finalizeTufRootChanges(newCiRoot, newProdRoot)

	fmt.Println("= Uploading new TUF root")
	tmpFile := saveTempTufCreds(keysFile, creds)
	err = api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil, nil)
	handleTufRootUpdatesUpload(tmpFile, keysFile, err)
}

func addOfflineRootKey(root *client.AtsTufRoot, creds OfflineCreds, keyType TufKeyType) {
	oldKids := root.Signed.Roles["root"].KeyIDs
	subcommands.DieNotNil(checkNoTufSigner(root, creds, oldKids))

	kp := genTufKeyPair(keyType)
	addOfflineTufKey(root, "root", kp, oldKids, creds)
	fmt.Println("= New root keyid:", kp.signer.Id)
}

func addOfflineTargetsKey(root *client.AtsTufRoot, creds OfflineCreds, keyType TufKeyType, onlineTargetsId string) {
	oldKids := root.Signed.Roles["targets"].KeyIDs
	if len(oldKids) > 1 {
		subcommands.DieNotNil(checkNoTufSigner(root, creds, subcommands.SliceRemove(oldKids, onlineTargetsId)))
	}

	kp := genTufKeyPair(keyType)
	addOfflineTufKey(root, "targets", kp, oldKids, creds)
	fmt.Println("= New targets keyid:", kp.signer.Id)
}
