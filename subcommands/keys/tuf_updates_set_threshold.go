package keys

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	setCmd := &cobra.Command{
		Use:   "set-threshold --role=root|targets <num>",
		Short: "Set signature threshold for the TUF root or production targets for your Factory",
		Long: `Set signature threshold for the TUF root or production targets for your Factory

For the TUF root the signature threshold is set for both CI and production roles (default 1).
For the TUF targets the signature threshold is only set for the production role (default 2).
Signature threshold for the CI TUF targets is always set to 1.

Make sure to add enough offline signing keys, and add enough signatures to satisfy the new signature threshold.
`,
		Args: cobra.ExactArgs(1),
		Run:  doTufUpdatesSetThreshold,
	}
	setCmd.Flags().StringP("role", "r", "", "TUF role name, supported: Root, Targets.")
	_ = setCmd.MarkFlagRequired("role")
	setCmd.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	setCmd.Flags().BoolP("sign", "s", false, "Sign the new TUF root using the offline root keys.")
	setCmd.Flags().StringP("keys", "k", "", "Path to <tuf-root-keys.tgz> used to sign TUF root.")
	_ = setCmd.MarkFlagFilename("keys")
	setCmd.MarkFlagsRequiredTogether("sign", "keys")
	tufUpdatesCmd.AddCommand(setCmd)
}

func doTufUpdatesSetThreshold(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	shouldSign, _ := cmd.Flags().GetBool("sign")
	keysFile, _ := cmd.Flags().GetString("keys")
	txid, _ := cmd.Flags().GetString("txid")
	roleName, _ := cmd.Flags().GetString("role")
	roleName = ParseTufRoleNameOffline(roleName)
	threshold, err := strconv.Atoi(args[0])
	subcommands.DieNotNil(err, fmt.Sprintf("Threshold argument must be an integer: %s", args[0]))

	var minThreshold = map[string]int{tufRoleNameRoot: 1, tufRoleNameTargets: 2}
	if threshold < minThreshold[roleName] {
		subcommands.DieNotNil(fmt.Errorf(
			"Threshold for TUF %s must be greater than %d: %d", roleName, minThreshold[roleName], threshold))
	}

	var updates client.TufRootUpdates
	updates, err = api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	curCiRoot, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)

	switch roleName {
	case tufRoleNameRoot:
		newCiRoot.Signed.Roles["root"].Threshold = threshold
	case tufRoleNameTargets:
		// For targets, only the production role should be modified
		if newProdRoot == nil {
			newProdRoot = genProdTufRoot(newCiRoot)
		}
		newProdRoot.Signed.Roles["targets"].Threshold = threshold
	default:
		panic(fmt.Errorf("Unexpected role name: %s", roleName))
	}
	newCiRoot, newProdRoot = finalizeTufRootChanges(newCiRoot, newProdRoot)

	if shouldSign {
		creds, err := GetOfflineCreds(keysFile)
		subcommands.DieNotNil(err)
		signNewTufRoot(curCiRoot, newCiRoot, newProdRoot, creds)
	}

	fmt.Println("= Uploading new TUF root")
	subcommands.DieNotNil(api.TufRootUpdatesPut(factory, txid, newCiRoot, newProdRoot, nil))
}
