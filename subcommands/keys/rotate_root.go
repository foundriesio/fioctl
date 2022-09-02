package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	doRootSync      bool
	initialRotation bool
	changeReason    string
)

func init() {
	rotate := &cobra.Command{
		Use:     "rotate-root <offline key archive>",
		Aliases: []string{"rotate"},
		Short:   "Rotate root signing key used by the Factory",
		Run:     doRotateRoot,
		Args:    cobra.ExactArgs(1),
	}
	rotate.Flags().BoolVarP(&doRootSync, "sync-prod", "", false, "Make sure production root.json is up-to-date and exit")
	rotate.Flags().BoolVarP(&initialRotation, "initial", "", false, "Used for the first customer rotation. The command will download the initial root key")
	rotate.Flags().StringVarP(&changeReason, "changelog", "m", "", "Reason for doing rotation. Saved in root metadata for tracking change history")
	rotate.Flags().StringP("key-type", "k", tufKeyTypeNameRSA, "Key type, supported: Ed25519, RSA (default).")
	cmd.AddCommand(rotate)
}

func doRotateRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	keyTypeStr, _ := cmd.Flags().GetString("key-type")
	keyType := ParseTufKeyType(keyTypeStr)
	credsFile := args[0]

	var creds OfflineCreds

	user, err := api.UserAccessDetails(factory, "self")
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	if initialRotation {
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
		subcommands.AssertWritable(credsFile)
		var err error
		creds, err = GetOfflineCreds(credsFile)
		subcommands.DieNotNil(err)
	}

	if doRootSync {
		subcommands.DieNotNil(syncProdRoot(factory, *root, creds))
		return
	}

	curPk, err := findRoot(*root, creds)
	subcommands.DieNotNil(err)
	fmt.Println("= Current root:", curPk.Id)

	// A rotation is pretty easy:
	// 1. change the who's listed as the root key: "swapRootKey"
	// 2. sign the new root.json with both the old and new root

	newPk, newCreds := swapRootKey(root, curPk.Id, creds, keyType)
	fmt.Println("= New root:", newPk.Id)
	root.Signed.Reason = &client.RootChangeReason{
		PolisId:   user.PolisId,
		Message:   changeReason,
		Timestamp: time.Now(),
	}

	fmt.Println("= Resigning root.json")
	signers := []TufSigner{*curPk, newPk}
	RemoveUnusedKeys(root)
	subcommands.DieNotNil(SignRoot(root, signers...))

	tufRootPost(factory, credsFile, root, newCreds)
}

func tufRootPost(factory, credsFile string, root *client.AtsTufRoot, creds OfflineCreds) {
	bytes, err := json.Marshal(root)
	subcommands.DieNotNil(err)

	// Create a backup before we try and commit this:
	tmpCreds := SaveTempCreds(credsFile, creds)

	fmt.Println("= Uploading rotated root")
	body, err := api.TufRootPost(factory, bytes)
	if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 409 {
		fmt.Println("ERROR: Your production root role is out of sync. Please run `fioctl keys rotate-root --sync-prod` to fix this.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println("\nERROR: ", err)
		fmt.Println(body)
		fmt.Println("A temporary copy of the new root was saved:", tmpCreds)
		fmt.Println("Before deleting this please ensure your factory isn't configured with this new key")
		os.Exit(1)
	}
	if err := os.Rename(tmpCreds, credsFile); err != nil {
		fmt.Println("\nERROR: Unable to update offline creds file.", err)
		fmt.Println("Temp copy still available at:", tmpCreds)
		fmt.Println("This temp file contains your new factory root private key. You must copy this file.")
	}

	// backfill this new key
	subcommands.DieNotNil(syncProdRoot(factory, *root, creds))
}

func swapRootKey(
	root *client.AtsTufRoot, curid string, creds OfflineCreds, keyType TufKeyType,
) (TufSigner, OfflineCreds) {
	kp := GenKeyPair(keyType)
	root.Signed.Keys[kp.signer.Id] = kp.atsPub
	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Roles["root"].KeyIDs = []string{kp.signer.Id}
	root.Signed.Version += 1

	base := "tufrepo/keys/fioctl-root-" + kp.signer.Id
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return kp.signer, creds
}

func findRoot(root client.AtsTufRoot, creds OfflineCreds) (*TufSigner, error) {
	kid := root.Signed.Roles["root"].KeyIDs[0]
	pub := root.Signed.Keys[kid].KeyValue.Public
	return FindSigner(kid, pub, creds)
}

func syncProdRoot(factory string, root client.AtsTufRoot, creds OfflineCreds) error {
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
	if err := SignRoot(&root, signers...); err != nil {
		return err
	}

	if len(root.Signed.Roles["targets"].KeyIDs) > 1 &&
		!subcommands.IsSliceSetEqual(
			root.Signed.Roles["targets"].KeyIDs,
			prevRoot.Signed.Roles["targets"].KeyIDs,
		) {
		//subcommands.DieNotNil(fmt.Errorf("HERE"))
		onlineTargetId, err := findOnlineTargetId(factory, root, creds)
		if err != nil {
			return err
		}
		err = resignProdTargets(factory, &root, onlineTargetId, creds)
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
