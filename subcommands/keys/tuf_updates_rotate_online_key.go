package keys

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rotate := &cobra.Command{
		Use:   "rotate-online-key --role targets|snapshot|timestamp [--txid=<txid>]",
		Short: "Stage rotation of the online TUF signing key for the Factory",
		Long: `Stage rotation of the online TUF signing key for the Factory.

The new online signing key will be used in both CI and production TUF root.

When you rotate the TUF online signing key:
- if there are CI or production targets in your factory, they are re-signed using the new key.
- if there is an active wave in your factory, the TUF online key rotation is not allowed.
- the new wave cannot be created until you apply the online keys rotation.

When you apply the online key rotation, these features are temporarily disabled until it succeeds:
- new CI targets upload (including the targets upload during CI builds).
- automatic re-signing of expired TUF roles using online keys (both CI and production targets).`,
		Example: `
- Rotate online TUF targets key and re-sign the new TUF root:
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
	rotate.Flags().StringP("key-type", "y", tufKeyTypeNameRSA, "Key type, supported: Ed25519, RSA (default).")
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
	_, _ = checkTufRootUpdatesStatus(updates, true)

	fmt.Println("= Generating new online TUF keys")
	subcommands.DieNotNil(api.TufRootUpdatesGenerateOnlineKeys(
		factory, txid, keyType.Name(), roleNames,
	))

	if shouldSign {
		creds, err := GetOfflineCreds(keysFile)
		subcommands.DieNotNil(err)

		updates, err := api.TufRootUpdatesGet(factory)
		subcommands.DieNotNil(err)

		curCiRoot, newCiRoot := checkTufRootUpdatesStatus(updates, true)
		newProdRoot := genProdTufRoot(newCiRoot)
		signNewTufRoot(curCiRoot, newCiRoot, newProdRoot, creds)

		fmt.Println("= Uploading new TUF root")
		subcommands.DieNotNil(api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil))
	}
}
