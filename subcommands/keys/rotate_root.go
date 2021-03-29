package keys

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	canonical "github.com/docker/go/canonical/json"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doRootSync bool

func init() {
	rotate := &cobra.Command{
		Use:     "rotate-root <offline key archive>",
		Aliases: []string{"rotate"},
		Short:   "Rotate root signing key used by the Factory",
		Run:     doRotateRoot,
		Args:    cobra.ExactArgs(1),
	}
	subcommands.RequireFactory(rotate)
	rotate.Flags().BoolVarP(&doRootSync, "sync-prod", "", false, "Make sure production root.json is up-to-date and exit")
	cmd.AddCommand(rotate)
}

func doRotateRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	assertWritable(credsFile)
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	if doRootSync {
		subcommands.DieNotNil(syncProdRoot(factory, *root, creds))
		return
	}

	curid, curPk, err := findRoot(*root, creds)
	fmt.Println("= Current root:", curid)
	subcommands.DieNotNil(err)

	// A rotation is pretty easy:
	// 1. change the who's listed as the root key: "swapRootKey"
	// 2. sign the new root.json with both the old and new root

	newid, newPk, newCreds := swapRootKey(root, curid, creds)
	fmt.Println("= New root:", newid)

	fmt.Println("= Resigning root.json")
	signers := []TufSigner{
		{Id: curid, Key: curPk},
		{Id: newid, Key: newPk},
	}
	removeUnusedKeys(root)
	subcommands.DieNotNil(signRoot(root, signers...))

	tufRootPost(factory, credsFile, root, newCreds)
}

func tufRootPost(factory, credsFile string, root *client.AtsTufRoot, creds OfflineCreds) {
	bytes, err := json.Marshal(root)
	subcommands.DieNotNil(err)

	// Create a backup before we try and commit this:
	tmpCreds := saveTempCreds(credsFile, creds)

	fmt.Println("= Uploading rotated root")
	body, err := api.TufRootPost(factory, bytes)
	if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 409 {
		fmt.Println("ERROR: Your production root role is out of sync. Please run `fioctl rotate root --sync-prod` to fix this.")
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

func removeUnusedKeys(root *client.AtsTufRoot) {
	var inuse []string
	for _, role := range root.Signed.Roles {
		inuse = append(inuse, role.KeyIDs...)
	}
	// we also have to be careful to not loose the extra root key when doing
	// a root key rotation
	for _, sig := range root.Signatures {
		inuse = append(inuse, sig.KeyID)
	}

	for k := range root.Signed.Keys {
		// is k in inuse?
		found := false
		for _, val := range inuse {
			if k == val {
				found = true
				break
			}
		}
		if !found {
			fmt.Println("= Removing unused key:", k)
			delete(root.Signed.Keys, k)
		}
	}
}

func signRoot(root *client.AtsTufRoot, signers ...TufSigner) error {
	bytes, err := canonical.MarshalCanonical(root.Signed)
	if err != nil {
		return err
	}
	signatures, err := SignMeta(bytes, signers...)
	if err != nil {
		return err
	}
	root.Signatures = signatures
	return nil
}

type keypair struct {
	rsaPriv      *rsa.PrivateKey
	atsPriv      client.AtsKey
	atsPrivBytes []byte

	atsPub      client.AtsKey
	atsPubBytes []byte

	keyid string
}

func genKeyPair() keypair {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	subcommands.DieNotNil(err)

	var privBytes []byte = x509.MarshalPKCS1PrivateKey(pk)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	priv := client.AtsKey{
		KeyType:  "RSA",
		KeyValue: client.AtsKeyVal{Private: string(pem.EncodeToMemory(block))},
	}
	atsPrivBytes, err := json.Marshal(priv)
	subcommands.DieNotNil(err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	subcommands.DieNotNil(err)
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pub := client.AtsKey{
		KeyType:  "RSA",
		KeyValue: client.AtsKeyVal{Public: string(pem.EncodeToMemory(block))},
	}
	atsPubBytes, err := json.Marshal(pub)
	subcommands.DieNotNil(err)

	id, err := pub.KeyID()
	subcommands.DieNotNil(err)

	return keypair{
		atsPriv:      priv,
		atsPrivBytes: atsPrivBytes,
		atsPub:       pub,
		atsPubBytes:  atsPubBytes,
		keyid:        id,
		rsaPriv:      pk,
	}
}

func swapRootKey(root *client.AtsTufRoot, curid string, creds OfflineCreds) (string, *rsa.PrivateKey, OfflineCreds) {
	kp := genKeyPair()
	root.Signed.Keys[kp.keyid] = kp.atsPub
	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Roles["root"].KeyIDs = []string{kp.keyid}
	root.Signed.Version += 1

	base := "tufrepo/keys/fioctl-root-" + kp.keyid
	creds[base+".pub"] = kp.atsPubBytes
	creds[base+".sec"] = kp.atsPrivBytes
	return kp.keyid, kp.rsaPriv, creds
}

func assertWritable(path string) {
	st, err := os.Stat(path)
	subcommands.DieNotNil(err)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, st.Mode())
	if err != nil {
		fmt.Println("ERROR: File is not writeable:", path)
		os.Exit(1)
	}
	f.Close()
}

func saveTempCreds(credsFile string, creds OfflineCreds) string {
	path := credsFile + ".tmp"
	if _, err := os.Stat(path); err == nil {
		subcommands.DieNotNil(fmt.Errorf(`Backup file exists: %s
This file may be from a previous failed key rotation and include critical data.
Please move this file somewhere safe before re-running this command.`,
			path,
		))
	}

	file, err := os.Create(path)
	subcommands.DieNotNil(err)
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, val := range creds {
		header := &tar.Header{
			Name: name,
			Size: int64(len(val)),
		}
		subcommands.DieNotNil(tarWriter.WriteHeader(header))
		_, err := tarWriter.Write(val)
		subcommands.DieNotNil(err)
	}
	return path
}

func findRoot(root client.AtsTufRoot, creds OfflineCreds) (string, *rsa.PrivateKey, error) {
	kid := root.Signed.Roles["root"].KeyIDs[0]
	pub := root.Signed.Keys[kid].KeyValue.Public
	key, err := FindPrivKey(pub, creds)
	return kid, key, err
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
		pkey, err := FindPrivKey(key.KeyValue.Public, creds)
		if err != nil {
			return err
		}
		signers = append(signers, TufSigner{
			Id:  sig.KeyID,
			Key: pkey,
		})
	}
	if err := signRoot(&root, signers...); err != nil {
		return err
	}

	if len(root.Signed.Roles["targets"].KeyIDs) > 1 &&
		!sliceSetEqual(root.Signed.Roles["targets"].KeyIDs, prevRoot.Signed.Roles["targets"].KeyIDs) {
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

func sliceSetEqual(first, second []string) bool {
	firstMap := make(map[string]int, len(first))
	for _, val := range first {
		firstMap[val] = 1
	}
	for _, val := range second {
		if _, ok := firstMap[val]; !ok {
			return false
		}
	}
	return true
}
