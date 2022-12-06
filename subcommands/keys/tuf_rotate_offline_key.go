package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	canonical "github.com/docker/go/canonical/json"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
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
	rotate.Flags().StringP("targets-keys", "K", "", "Path to <offline-targets-creds.tgz> used to sign prod & wave TUF targets.")
	_ = rotate.MarkFlagFilename("targets-keys")
	rotate.Flags().BoolP("first-time", "", false, "Used for the first customer rotation. The command will download the initial root key.")
	rotate.Flags().StringP("key-type", "y", tufKeyTypeNameRSA, "Key type, supported: Ed25519, RSA (default).")
	rotate.Flags().StringP("changelog", "m", "", "Reason for doing rotation. Saved in root metadata for tracking change history.")
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
	legacyRotateRoot.Flags().BoolP("initial", "", false, "Used for the first customer rotation. The command will download the initial root key")
	legacyRotateRoot.Flags().StringP("changelog", "m", "", "Reason for doing rotation. Saved in root metadata for tracking change history")
	legacyRotateRoot.Flags().StringP("key-type", "y", tufKeyTypeNameRSA, "Key type, supported: Ed25519, RSA (default).")
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
	legacyRotateTargets.Flags().StringP("key-type", "y", tufKeyTypeNameRSA, "Key type, supported: Ed25519, RSA (default).")
	legacyRotateTargets.Flags().StringP("changelog", "m", "", "Reason for doing rotation. Saved in root metadata for tracking change history.")
	cmd.AddCommand(legacyRotateTargets)

	tempSyncProdRoot := &cobra.Command{
		Use:    "sync-prod-root",
		Short:  "Make sure production root.json is up-to-date after a failed TUF key rotation and exit",
		Hidden: true,
		Run:    doSyncProdRoot,
	}
	tempSyncProdRoot.Flags().StringP("keys", "k", "", "Path to <offline-creds.tgz> used to sign TUF root.")
	_ = tempSyncProdRoot.MarkFlagRequired("keys")
	_ = tempSyncProdRoot.MarkFlagFilename("keys")
	tempSyncProdRoot.Flags().StringP("targets-keys", "K", "", "Path to <offline-targets-creds.tgz> used to sign prod & wave TUF targets.")
	_ = tempSyncProdRoot.MarkFlagFilename("targets-keys")
	tufCmd.AddCommand(tempSyncProdRoot)
}

func doSyncProdRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile, _ := cmd.Flags().GetString("keys")
	targetsCredsFile, _ := cmd.Flags().GetString("targets-keys")
	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)
	if targetsCredsFile == "" {
		subcommands.DieNotNil(syncProdRoot(factory, root, creds, creds))
	} else {
		targetsCreds, err := GetOfflineCreds(targetsCredsFile)
		subcommands.DieNotNil(err)
		subcommands.DieNotNil(syncProdRoot(factory, root, creds, targetsCreds))
	}
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

func swapRootKey(
	root *client.AtsTufRoot, creds OfflineCreds, keyType TufKeyType,
) (*TufSigner, OfflineCreds) {
	kp := genTufKeyPair(keyType)
	root.Signed.Keys[kp.signer.Id] = kp.atsPub
	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Roles["root"].KeyIDs = []string{kp.signer.Id}

	base := "tufrepo/keys/fioctl-root-" + kp.signer.Id
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return &kp.signer, creds
}

func syncProdRoot(factory string, root *client.AtsTufRoot, creds, targetsCreds OfflineCreds) error {
	fmt.Println("= Populating production root version")

	if root.Signed.Version == 1 {
		return fmt.Errorf("Unexpected error: production root version 1 can only be generated on server side")
	}

	prevRoot, err := api.TufRootGetVer(factory, root.Signed.Version-1)
	if err != nil {
		return err
	}

	// Bump the threshold
	root.Signed.Roles["targets"].Threshold = 2

	// Sign with the same keys used for the ci copy
	var signers []TufSigner
	for _, sig := range root.Signatures {
		key, ok := root.Signed.Keys[sig.KeyID]
		if !ok {
			key = prevRoot.Signed.Keys[sig.KeyID]
		}
		signer, err := FindTufSigner(sig.KeyID, key.KeyValue.Public, creds)
		if err != nil {
			return err
		}
		signers = append(signers, *signer)
	}
	if err := signTufRoot(root, signers...); err != nil {
		return err
	}

	if len(root.Signed.Roles["targets"].KeyIDs) > 1 &&
		!subcommands.IsSliceSetEqual(
			root.Signed.Roles["targets"].KeyIDs,
			prevRoot.Signed.Roles["targets"].KeyIDs,
		) {
		onlineTargetsId, err := findOnlineTargetsId(factory, *root)
		if err != nil {
			return err
		}
		root.TargetsSignatures, err = resignProdTargets(factory, root, onlineTargetsId, targetsCreds)
		if err != nil {
			return err
		}
	}

	bytes, err := json.Marshal(root)
	if err != nil {
		return err
	}
	_, err = api.TufProdRootPost(factory, bytes)
	if err != nil {
		return err
	}
	return nil
}

func findOnlineTargetsId(factory string, root client.AtsTufRoot) (string, error) {
	onlinePub, err := api.TufTargetsOnlineKey(factory)
	subcommands.DieNotNil(err)
	for _, keyid := range root.Signed.Roles["targets"].KeyIDs {
		pub := root.Signed.Keys[keyid].KeyValue.Public
		if pub == onlinePub.KeyValue.Public {
			return keyid, nil
		}
	}
	return "", errors.New("Unable to find online target key for factory")
}

func replaceOfflineTargetsKey(
	root *client.AtsTufRoot, onlineTargetsId string, creds OfflineCreds, keyType TufKeyType,
) (*TufSigner, OfflineCreds) {
	kp := genTufKeyPair(keyType)
	root.Signed.Keys[kp.signer.Id] = kp.atsPub
	root.Signed.Roles["targets"].KeyIDs = []string{onlineTargetsId, kp.signer.Id}
	root.Signed.Roles["targets"].Threshold = 1

	base := "tufrepo/keys/fioctl-targets-" + kp.signer.Id
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return &kp.signer, creds
}

func resignProdTargets(
	factory string, root *client.AtsTufRoot, onlineTargetsId string, creds OfflineCreds,
) (map[string][]tuf.Signature, error) {
	targetsMap, err := api.ProdTargetsList(factory, false)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch production targets: %w", err)
	} else if targetsMap == nil {
		return nil, nil
	}

	var signers []TufSigner
	for _, kid := range root.Signed.Roles["targets"].KeyIDs {
		if kid == onlineTargetsId {
			continue
		}
		pub := root.Signed.Keys[kid].KeyValue.Public
		signer, err := FindTufSigner(kid, pub, creds)
		if err != nil {
			return nil, fmt.Errorf("Failed to find private key for %s: %w", kid, err)
		}
		signers = append(signers, *signer)
	}

	signatureMap := make(map[string][]tuf.Signature)
	for tag, targets := range targetsMap {
		bytes, err := canonical.MarshalCanonical(targets.Signed)
		if err != nil {
			return nil, fmt.Errorf("Failed to marshal targets for tag %s: %w", tag, err)
		}
		signatures, err := SignTufMeta(bytes, signers...)
		if err != nil {
			return nil, fmt.Errorf("Failed to re-sign targets for tag %s: %w", tag, err)
		}
		signatureMap[tag] = signatures
	}
	return signatureMap, nil
}
