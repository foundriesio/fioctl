package keys

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rotate := &cobra.Command{
		Use:   "rotate-online-key --role targets|snapshot|timestamp [--txid=<txid>]",
		Short: "Stage rotation of the online TUF signing key for the Factory",
		Long: `Stage rotation of the online TUF signing key for the Factory.

The new online signing key will be used in both CI and production TUF root.

When you rotate the TUF online signing key:
- Any CI or production Targets are re-signed using the new key.
- If there is an active Wave, the TUF online key rotation is not allowed.
- The new Wave cannot be created until you apply the online keys rotation.

When you apply the online key rotation, these features are temporarily disabled until it succeeds:
- new CI targets upload (including the Targets upload during CI builds).
- automatic re-signing of expired TUF roles using online keys (both CI and production Targets).`,
		Example: `
- Rotate online TUF Targets key and re-sign the new TUF root:
  fioctl keys tuf updates rotate-online-key \
    --txid=abc --role=targets --keys=tuf-root-keys.tgz --sign
- Rotate all online TUF keys explicitly specifying new key type (and signing algorithm):
  fioctl keys tuf updates rotate-online-key \
    --txid=abc --role=targets,snapshot,timestamp --key-type=ed25519`,
		Run: doTufUpdatesRotateOnlineKey,
	}
	rotate.Flags().StringSliceP("role", "r", nil, "TUF role name, supported: Targets, Snapshot, Timestamp.")
	_ = rotate.MarkFlagRequired("role")
	rotate.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	rotate.Flags().StringP("keys", "k", "", "Path to <tuf-root-keys.tgz> used to sign TUF root.")
	_ = rotate.MarkFlagFilename("keys")
	rotate.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519, "Key type, supported: Ed25519, RSA.")
	rotate.Flags().BoolP("sign", "s", false, "Sign the new TUF root using the offline root keys.")
	rotate.MarkFlagsRequiredTogether("sign", "keys")
	tufUpdatesCmd.AddCommand(rotate)
}

func doTufUpdatesRotateOnlineKey(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")
	roleNames, _ := cmd.Flags().GetStringSlice("role")
	for idx, roleName := range roleNames {
		roleNames[idx] = strings.ToLower(ParseTufRoleNameOnline(roleName))
	}
	keyTypeStr, _ := cmd.Flags().GetString("key-type")
	keyType := ParseTufKeyType(keyTypeStr)
	keysFile, _ := cmd.Flags().GetString("keys")
	shouldSign, _ := cmd.Flags().GetBool("sign")

	// Preliminary check to give a more verbose error message before requesting to generate new keys
	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)
	_, _, _ = checkTufRootUpdatesStatus(updates, true)

	fmt.Println("= Generating new online TUF keys")
	subcommands.DieNotNil(api.TufRootUpdatesGenerateOnlineKeys(
		factory, txid, keyType.Name(), roleNames,
	))

	updates, err = api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err, "Failed to fetch new online TUF keys")
	for _, roleName := range []string{tufRoleNameTargets, tufRoleNameSnapshot, tufRoleNameTimestamp} {
		roleName = strings.ToLower(roleName)
		if slices.Contains(roleNames, roleName) {
			fmt.Printf("= New online %s keyid: %s\n", roleName, updates.Updated.OnlineKeys[roleName])
		}
	}

	if shouldSign {
		creds, err := GetOfflineCreds(keysFile)
		subcommands.DieNotNil(err)

		curCiRoot, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)
		newCiRoot, newProdRoot = finalizeTufRootChanges(newCiRoot, newProdRoot)
		signNewTufRoot(curCiRoot, newCiRoot, newProdRoot, creds)

		fmt.Println("= Uploading new TUF root")
		subcommands.DieNotNil(api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil, nil))
	}
}
