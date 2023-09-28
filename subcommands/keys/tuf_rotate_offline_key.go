package keys

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

const (
	tufCmdAnnotation          = "tuf-cmd"
	tufCmdRotateOfflineKey    = "rotate-offline-key"
	tufCmdRotateRootLegacy    = "rotate-root"
	tufCmdRotateTargetsLegacy = "rotate-targets"
)

func init() {
	rotate := &cobra.Command{
		Use:   "rotate-offline-key --role root|targets --keys=<offline-creds.tgz>",
		Short: "Rotate the offline TUF signing key for the Factory",
		Long: `Rotate the TUF root or TUF targets offline signing key for the Factory.

The new signing key is rotated in both CI and production TUF root transactionally.

When you rotate the TUF targets offline signing key:
- if there are production targets in your factory, they are re-signed using the new key.
- if there is an active wave in your factory, the TUF targets rotation is not allowed.`,
		Example: `
  # Take ownership of TUF root and targets keys for a new factory, keep them in separate files:
  fioctl keys tuf rotate-offline-key --role=root \
    --keys=offline-tuf-root-keys.tgz --first-time
  fioctl keys tuf rotate-offline-key --role=targets \
    --keys=offline-tuf-root-keys.tgz --targets-keys=offline-tuf-targets-keys.tgz

  # Rotate offline TUF targets key using the Ed25519 elliptic curve to generate a new key pair:
  fioctl keys tuf rotate-offline-key --role=targets --key-type=ed25519 \
    --keys=offline-tuf-root-keys.tgz --targets-keys=offline-tuf-targets-keys.tgz`,
		Run:         doRotateOfflineKey,
		Annotations: map[string]string{tufCmdAnnotation: tufCmdRotateOfflineKey},
	}
	rotate.Flags().StringP("role", "r", "", "TUF role name, supported: Root, Targets.")
	_ = rotate.MarkFlagRequired("role")
	rotate.Flags().StringP("keys", "k", "", "Path to <offline-creds.tgz> used to sign TUF root.")
	_ = rotate.MarkFlagRequired("keys")
	_ = rotate.MarkFlagFilename("keys")
	rotate.Flags().StringP("targets-keys", "K", "",
		"Path to <offline-targets-creds.tgz> used to sign prod & wave TUF targets.")
	_ = rotate.MarkFlagFilename("targets-keys")
	rotate.Flags().BoolP("first-time", "", false, "Used for the first customer rotation. "+
		"The command will download the initial root key.")
	rotate.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519, "Key type, supported: Ed25519, RSA.")
	rotate.Flags().StringP("changelog", "m", "", "Reason for doing rotation. "+
		"Saved in root metadata for tracking change history.")
	tufCmd.AddCommand(rotate)

	legacyRotateRoot := &cobra.Command{
		Use:     "rotate-root <offline key archive>",
		Aliases: []string{"rotate"},
		Short:   "Rotate the offline target signing key for the Factory",
		Deprecated: `it has moved to a new place.
Instead, please, use a new approach to rotate TUF root key:
  fioctl keys tuf rotate-offline-key --role=root --keys=<offline-creds.tgz>
`,
		Hidden:      true,
		Run:         doRotateOfflineKey,
		Annotations: map[string]string{tufCmdAnnotation: tufCmdRotateRootLegacy},
		Args:        cobra.ExactArgs(1),
	}
	legacyRotateRoot.Flags().BoolP("initial", "", false,
		"Used for the first customer rotation. The command will download the initial root key")
	legacyRotateRoot.Flags().StringP("changelog", "m", "",
		"Reason for doing rotation. Saved in root metadata for tracking change history")
	legacyRotateRoot.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519,
		"Key type, supported: Ed25519, RSA.")
	cmd.AddCommand(legacyRotateRoot)

	legacyRotateTargets := &cobra.Command{
		Use:   "rotate-targets <offline-creds.tgz>",
		Short: "Rotate the offline target signing key for the Factory",
		Long: `Rotate the offline target signing key for the Factory.

If there are any production targets in your factory - they are re-signed using the new key.
This command is not allowed if there is an active wave in your factory.`,
		Deprecated: `it has moved to a new place.
Instead, please, use a new approach to rotate TUF targets key:
  fioctl keys tuf rotate-offline-key --role=targets
    --keys=<offline-creds.tgz> [--targets-keys=<offline-targets-creds.tgz>]
`,
		Hidden:      true,
		Run:         doRotateOfflineKey,
		Annotations: map[string]string{tufCmdAnnotation: tufCmdRotateTargetsLegacy},
		Args:        cobra.ExactArgs(1),
	}
	legacyRotateTargets.Flags().StringP("key-type", "y", tufKeyTypeNameEd25519, "Key type, supported: Ed25519, RSA.")
	legacyRotateTargets.Flags().StringP("changelog", "m", "",
		"Reason for doing rotation. Saved in root metadata for tracking change history.")
	cmd.AddCommand(legacyRotateTargets)
}

func doRotateOfflineKey(cmd *cobra.Command, args []string) {
	isTufUpdatesShortcut = true
	var (
		roleName, credsFile string
		targetsCredsFile    string
		firstTime           bool
	)
	keyType, _ := cmd.Flags().GetString("key-type")
	ParseTufKeyType(keyType) // fails on error
	changelog, _ := cmd.Flags().GetString("changelog")
	cmdName := cmd.Annotations[tufCmdAnnotation]
	switch cmdName {
	case tufCmdRotateOfflineKey:
		roleName, _ = cmd.Flags().GetString("role")
		roleName = ParseTufRoleNameOffline(roleName)
		credsFile, _ = cmd.Flags().GetString("keys")
		targetsCredsFile, _ = cmd.Flags().GetString("targets-keys")
		firstTime, _ = cmd.Flags().GetBool("first-time")
		if firstTime && roleName != tufRoleNameRoot {
			subcommands.DieNotNil(errors.New("The --first-time option is only valid for the first TUF root key rotation."))
		}
		if targetsCredsFile != "" && roleName != tufRoleNameTargets {
			subcommands.DieNotNil(errors.New("The --targets-keys option is only valid for the TUF targets key rotation."))
		}
	case tufCmdRotateRootLegacy:
		roleName = tufRoleNameRoot
		credsFile = args[0]
		firstTime, _ = cmd.Flags().GetBool("initial")
	case tufCmdRotateTargetsLegacy:
		roleName = tufRoleNameTargets
		credsFile = args[0]
		firstTime = false
	default:
		panic(fmt.Errorf("Unexpected command: %s", cmdName))
	}

	if changelog == "" {
		switch roleName {
		case tufRoleNameRoot:
			changelog = "Rotate TUF root offline signing key"
		case tufRoleNameTargets:
			changelog = "Rotate TUF targets offline signing key"
		default:
			panic(fmt.Errorf("Unexpected role name: %s", roleName))
		}
	}

	// Below the `tuf updates` subcommands are chained in a correct order.
	// Detach from the parent, so that command calls below use correct args.
	tufCmd.RemoveCommand(tufUpdatesCmd)

	fmt.Println("= Creating new TUF updates transaction")
	args = []string{"init", "-m", changelog}
	if firstTime {
		args = append(args, "--first-time", "-k", credsFile)
	}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	args = []string{"rotate-offline-key", "-r", roleName, "-k", credsFile, "-y", keyType, "-s"}
	if targetsCredsFile != "" {
		args = append(args, "-K", targetsCredsFile)
	}
	tufUpdatesCmd.SetArgs(args)
	subcommands.DieNotNil(tufUpdatesCmd.Execute())

	fmt.Println("= Applying staged TUF root changes")
	tufUpdatesCmd.SetArgs([]string{"apply"})
	subcommands.DieNotNil(tufUpdatesCmd.Execute())
}
