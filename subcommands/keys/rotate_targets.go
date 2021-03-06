package keys

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	canonical "github.com/docker/go/canonical/json"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	rotateTargets := &cobra.Command{
		Use:   "rotate-targets <offline-creds.tgz>",
		Short: "Rotate the offline target signing key for the Factory",
		Long: `Rotate the offline target signing key for the Factory.

If there are any production targets in your factory - they are re-signed using the new key.
This command is not allowed if there is an active wave in your factory.`,
		Run: doRotateTargets,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = subcommands.Login(cmd)
		},
		Args: cobra.ExactArgs(1),
	}
	subcommands.RequireFactory(rotateTargets)
	cmd.AddCommand(rotateTargets)
}

func doRotateTargets(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	assertWritable(credsFile)
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	// Target "rotation" works like this:
	// 1. Find the "online target key" - this the key used by CI, so we don't
	//    want to lose it.
	// 2. Generate a new key.
	// 3. Set these keys in root.json

	onlineTargetId, err := findOnlineTargetId(factory, *root, creds)
	subcommands.DieNotNil(err)

	rootid, rootPk, err := findRoot(*root, creds)
	fmt.Println("= Root keyid:", rootid)
	subcommands.DieNotNil(err)
	targetid, newCreds := replaceOfflineTargetKey(root, onlineTargetId, creds)
	fmt.Println("= New target:", targetid)
	removeUnusedKeys(root)
	subcommands.DieNotNil(signRoot(root, TufSigner{rootid, rootPk}))
	subcommands.DieNotNil(resignProdTargets(factory, root, onlineTargetId, creds))

	tufRootPost(factory, credsFile, root, newCreds)
}

func findOnlineTargetId(factory string, root client.AtsTufRoot, creds OfflineCreds) (string, error) {
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

func replaceOfflineTargetKey(root *client.AtsTufRoot, onlineTargetId string, creds OfflineCreds) (string, OfflineCreds) {
	kp := genKeyPair()
	root.Signed.Keys[kp.keyid] = kp.atsPub
	root.Signed.Roles["targets"].KeyIDs = []string{onlineTargetId, kp.keyid}
	root.Signed.Roles["targets"].Threshold = 1
	root.Signed.Version += 1

	base := "tufrepo/keys/fioctl-targets-" + kp.keyid
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return kp.keyid, creds
}

func resignProdTargets(
	factory string, root *client.AtsTufRoot, onlineTargetId string, creds OfflineCreds,
) error {
	targetsMap, err := api.ProdTargetsList(factory, false)
	if err != nil {
		return fmt.Errorf("Failed to fetch production targets: %w", err)
	} else if targetsMap == nil {
		return nil
	}

	signers := make([]TufSigner, 0)
	for _, kid := range root.Signed.Roles["targets"].KeyIDs {
		if kid == onlineTargetId {
			continue
		}
		pub := root.Signed.Keys[kid].KeyValue.Public
		pkey, err := FindPrivKey(pub, creds)
		if err != nil {
			return fmt.Errorf("Failed to find private key for %s: %w", kid, err)
		}
		signers = append(signers, TufSigner{Id: kid, Key: pkey})
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
