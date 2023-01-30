package keys

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rotate := &cobra.Command{
		Use:   "rotate-all-keys --keys=<offline-creds.tgz>",
		Short: "Rotate all online and offline TUF signing keys for the Factory",
		Long: `Rotate the following TUF keys for the Factory:
- offline root signing key;
- offline targets signing key;
- online targets signing key;
- online snapshot signing key;
- online timestamp signing key.

The new signing keys are rotated in both CI and production TUF root transactionally.

When you rotate all TUF signing leys:
- if there are CI or production targets in your factory, they are re-signed using the new keys.
- if there is an active wave in your factory, this command is not allowed.
- new CI targets upload is temporarily disabled for the duration of transaction.`,
		Example: `
Migrate an old factory to use Ed25519 key type for all TUF signing keys (online and offline):
  fioctl keys tuf rotate-all-keys --key-type=ed25519 \
    --keys=offline-tuf-root-keys.tgz --targets-keys=offline-tuf-targets-keys.tgz`,
		Run: doRotateAllKeys,
	}
	rotate.Flags().StringP("keys", "k", "", "Path to <offline-creds.tgz> used to sign TUF root.")
	_ = rotate.MarkFlagRequired("keys")
	_ = rotate.MarkFlagFilename("keys")
	rotate.Flags().StringP("targets-keys", "K", "", "Path to <offline-targets-creds.tgz> used to sign prod & wave TUF targets.")
	_ = rotate.MarkFlagFilename("targets-keys")
	rotate.Flags().BoolP("first-time", "", false, "Used for the first customer rotation. The command will download the initial root key.")
	rotate.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519, "Key type, supported: Ed25519, RSA.")
	rotate.Flags().StringP("changelog", "m", "", "Reason for doing rotation. Saved in root metadata for tracking change history.")
	tufCmd.AddCommand(rotate)
}

func doRotateAllKeys(cmd *cobra.Command, unusedArgs []string) {
	isTufUpdatesShortcut = true
	keyType, _ := cmd.Flags().GetString("key-type")
	ParseTufKeyType(keyType) // fails on error
	changelog, _ := cmd.Flags().GetString("changelog")
	if changelog == "" {
		changelog = "Rotate all TUF root signing keys"
	}
	firstTime, _ := cmd.Flags().GetBool("first-time")
	credsFile, _ := cmd.Flags().GetString("keys")
	targetsCredsFile, _ := cmd.Flags().GetString("targets-keys")
	if targetsCredsFile == "" {
		targetsCredsFile = credsFile
	}

	// Below the `tuf updates` subcommands are chained in a correct order.
	// Detach from the parent, so that command calls below use correct args.
	tufCmd.RemoveCommand(tufUpdatesCmd)

	fmt.Println("= Creating new TUF updates transaction")
	args := []string{"init", "-m", changelog}
	if firstTime {
		args = append(args, "--first-time", "-k", credsFile)
	}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	args = []string{"rotate-offline-key", "-r", "root", "-k", credsFile, "-y", keyType}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	args = []string{"rotate-offline-key", "-r", "targets", "-k", targetsCredsFile, "-y", keyType}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	args = []string{"rotate-online-key", "-r", "targets,snapshot,timestamp", "-y", keyType}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	args = []string{"sign", "-k", credsFile}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	fmt.Println("= Applying staged TUF root changes")
	tufUpdatesCmd.SetArgs([]string{"apply"})
	subcommands.DieNotNil(tufUpdatesCmd.Execute())
}
