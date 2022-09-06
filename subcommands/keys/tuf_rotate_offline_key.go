package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	var (
		roleName, credsFile string
		targetsCredsFile    string
		firstTime           bool
		creds, newCreds     OfflineCreds
		newKey              *TufSigner
		signers             []TufSigner
		err                 error
	)
	factory := viper.GetString("factory")
	keyTypeStr, _ := cmd.Flags().GetString("key-type")
	keyType := ParseTufKeyType(keyTypeStr)
	changeLog, _ := cmd.Flags().GetString("changelog")
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

	user, err := api.UserAccessDetails(factory, "self")
	subcommands.DieNotNil(err)
	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	if firstTime {
		if _, err := os.Stat(credsFile); err == nil {
			subcommands.DieNotNil(errors.New("Destination file exists. Please make sure you aren't accidentally overwriting another factory's keys"))
		}

		key, err := api.TufRootFirstKey(factory)
		subcommands.DieNotNil(err)

		pkid := root.Signed.Roles["root"].KeyIDs[0]
		pub := root.Signed.Keys[pkid]

		creds = make(OfflineCreds)
		bytes, err := json.Marshal(key)
		subcommands.DieNotNil(err)
		creds["tufrepo/keys/first-root.sec"] = bytes

		bytes, err = json.Marshal(pub)
		subcommands.DieNotNil(err)
		creds["tufrepo/keys/first-root.pub"] = bytes

		SaveCreds(credsFile, creds)
	} else {
		creds, err = GetOfflineCreds(credsFile)
		subcommands.DieNotNil(err)
	}

	rootKey, err := findRoot(root, creds)
	subcommands.DieNotNil(err)
	signers = append(signers, *rootKey)
	credsFileToSave := credsFile

	switch roleName {
	case tufRoleNameRoot:
		// A rotation is pretty easy:
		// 1. change the who's listed as the root key: "swapRootKey"
		// 2. sign the new root.json with both the old and new root
		subcommands.AssertWritable(credsFile)
		fmt.Println("= Current root keyid:", rootKey.Id)
		newKey, newCreds = swapRootKey(root, rootKey.Id, creds, keyType)
		fmt.Println("= New root keyid:", newKey.Id)
		signers = append(signers, *newKey)
		if changeLog == "" {
			changeLog = "Root role offline key rotation, new keyid: " + newKey.Id
		}

	case tufRoleNameTargets:
		// Target "rotation" works like this:
		// 1. Find the "online target key" - this the key used by CI, so we don't
		//    want to lose it.
		// 2. Generate a new key.
		// 3. Set these keys in root.json.
		// 4. Re-sign existing production targets.
		targetsCreds := creds
		if targetsCredsFile != "" {
			if _, err = os.Stat(targetsCredsFile); err == nil {
				targetsCreds, err = GetOfflineCreds(targetsCredsFile)
				subcommands.DieNotNil(err)
			} else if os.IsNotExist(err) {
				// Targets key rotation to a new file - just verify it is writeable
				targetsCreds = make(OfflineCreds, 0)
			} else {
				subcommands.DieNotNil(err)
			}
			subcommands.AssertWritable(targetsCredsFile)
			credsFileToSave = targetsCredsFile
		} else {
			subcommands.AssertWritable(credsFile)
		}

		onlineTargetsId, err := findOnlineTargetsId(factory, *root)
		subcommands.DieNotNil(err)
		newKey, newCreds = replaceOfflineTargetsKey(root, onlineTargetsId, targetsCreds, keyType)
		fmt.Println("= New target keyid:", newKey.Id)
		fmt.Println("= Resigning prod targets")
		subcommands.DieNotNil(resignProdTargets(factory, root, onlineTargetsId, newCreds))
		if changeLog == "" {
			changeLog = "Targets role offline key rotation, new keyid: " + newKey.Id
		}
	default:
		panic(fmt.Errorf("Unexpected role name: %s", roleName))
	}

	RemoveUnusedKeys(root)
	root.Signed.Reason = &client.RootChangeReason{
		PolisId:   user.PolisId,
		Message:   changeLog,
		Timestamp: time.Now(),
	}
	fmt.Println("= Resigning root.json")
	subcommands.DieNotNil(SignRoot(root, signers...))

	recoverySyncProdRootArgs := "--keys " + credsFile
	if targetsCredsFile != "" {
		recoverySyncProdRootArgs += " --targets-keys " + targetsCredsFile
	}
	tufRootPost(factory, credsFileToSave, recoverySyncProdRootArgs, root, newCreds)

	// backfill this new key
	switch roleName {
	case tufRoleNameRoot:
		// newCreds contains a new TUF root key; there is no need to resign production targets
		subcommands.DieNotNil(syncProdRoot(factory, root, newCreds, nil))
	case tufRoleNameTargets:
		// newCreds contains a new TUF targets key; creds contains existing TUF root key
		subcommands.DieNotNil(syncProdRoot(factory, root, creds, newCreds))
	}
}

func tufRootPost(
	factory, credsFileToSave, syncProdRootArgs string, root *client.AtsTufRoot, credsToSave OfflineCreds,
) {
	bytes, err := json.Marshal(root)
	subcommands.DieNotNil(err)

	// Create a backup before we try and commit this:
	tmpCreds := SaveTempCreds(credsFileToSave, credsToSave)

	fmt.Println("= Uploading rotated root")
	body, err := api.TufRootPost(factory, bytes)
	if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 409 {
		fmt.Printf(`ERROR: Your production root role is out of sync.
Please run a hidden "fioctl keys tuf sync-prod-root %s" command to fix this.`, syncProdRootArgs)
		os.Exit(1)
	} else if err != nil {
		fmt.Println("\nERROR: ", err)
		fmt.Println(body)
		fmt.Println("A temporary copy of the new root was saved:", tmpCreds)
		fmt.Println("Before deleting this please ensure your factory isn't configured with this new key")
		os.Exit(1)
	}
	if err := os.Rename(tmpCreds, credsFileToSave); err != nil {
		fmt.Println("\nERROR: Unable to update offline keys file.", err)
		fmt.Println("Temp copy still available at:", tmpCreds)
		fmt.Println("This temp file contains your new factory private key. You must copy this file.")
	}
}

func swapRootKey(
	root *client.AtsTufRoot, curid string, creds OfflineCreds, keyType TufKeyType,
) (*TufSigner, OfflineCreds) {
	kp := GenKeyPair(keyType)
	root.Signed.Keys[kp.signer.Id] = kp.atsPub
	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Roles["root"].KeyIDs = []string{kp.signer.Id}
	root.Signed.Version += 1

	base := "tufrepo/keys/fioctl-root-" + kp.signer.Id
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return &kp.signer, creds
}

func findRoot(root *client.AtsTufRoot, creds OfflineCreds) (*TufSigner, error) {
	kid := root.Signed.Roles["root"].KeyIDs[0]
	pub := root.Signed.Keys[kid].KeyValue.Public
	return FindSigner(kid, pub, creds)
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
		signer, err := FindSigner(sig.KeyID, key.KeyValue.Public, creds)
		if err != nil {
			return err
		}
		signers = append(signers, *signer)
	}
	if err := SignRoot(root, signers...); err != nil {
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
		err = resignProdTargets(factory, root, onlineTargetsId, targetsCreds)
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
	onlinePub, err := api.GetFoundriesTargetsKey(factory)
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
	kp := GenKeyPair(keyType)
	root.Signed.Keys[kp.signer.Id] = kp.atsPub
	root.Signed.Roles["targets"].KeyIDs = []string{onlineTargetsId, kp.signer.Id}
	root.Signed.Roles["targets"].Threshold = 1
	root.Signed.Version += 1

	base := "tufrepo/keys/fioctl-targets-" + kp.signer.Id
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return &kp.signer, creds
}

func resignProdTargets(
	factory string, root *client.AtsTufRoot, onlineTargetsId string, creds OfflineCreds,
) error {
	targetsMap, err := api.ProdTargetsList(factory, false)
	if err != nil {
		return fmt.Errorf("Failed to fetch production targets: %w", err)
	} else if targetsMap == nil {
		return nil
	}

	var signers []TufSigner
	for _, kid := range root.Signed.Roles["targets"].KeyIDs {
		if kid == onlineTargetsId {
			continue
		}
		pub := root.Signed.Keys[kid].KeyValue.Public
		signer, err := FindSigner(kid, pub, creds)
		if err != nil {
			return fmt.Errorf("Failed to find private key for %s: %w", kid, err)
		}
		signers = append(signers, *signer)
	}

	signatureMap := make(map[string][]tuf.Signature)
	for tag, targets := range targetsMap {
		bytes, err := canonical.MarshalCanonical(targets.Signed)
		if err != nil {
			return fmt.Errorf("Failed to marshal targets for tag %s: %w", tag, err)
		}
		signatures, err := SignMeta(bytes, signers...)
		if err != nil {
			return fmt.Errorf("Failed to re-sign targets for tag %s: %w", tag, err)
		}
		signatureMap[tag] = signatures
	}
	root.TargetsSignatures = signatureMap
	return nil
}
