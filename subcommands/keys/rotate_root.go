package keys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	canonical "github.com/docker/go/canonical/json"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"
)

type OfflineCreds map[string][]byte
type TufSigner struct {
	Id  string
	Key *rsa.PrivateKey
}

func init() {
	rotate := &cobra.Command{
		Use:     "rotate-root <offline key archive>",
		Aliases: []string{"rotate"},
		Short:   "Rotate root signing key used by the Factory",
		Run:     doRotateRoot,
		Args:    cobra.ExactArgs(1),
	}
	subcommands.RequireFactory(rotate)
	cmd.AddCommand(rotate)
}

func doRotateRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	assertWritable(credsFile)
	creds := getOfflineCreds(credsFile)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

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
	subcommands.DieNotNil(signRoot(root, signers...))

	bytes, err := json.Marshal(root)
	subcommands.DieNotNil(err)

	// Create a backup before we try and commit this:
	tmpCreds := saveTempCreds(credsFile, newCreds)

	fmt.Println("= Uploading rotated root")
	body, err := api.TufRootPost(factory, bytes)
	if err != nil {
		fmt.Println("\nERROR:", err)
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
	removeUnusedKeys(root)

	bytes, err := canonical.MarshalCanonical(root.Signed)
	if err != nil {
		return err
	}

	opts := rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}
	hashed := sha256.Sum256(bytes)

	root.Signatures = []tuf.Signature{}

	for _, signer := range signers {
		bytes, err = signer.Key.Sign(rand.Reader, hashed[:], &opts)
		if err != nil {
			return err
		}
		sig := tuf.Signature{
			KeyID:     signer.Id,
			Method:    "rsassa-pss-sha256",
			Signature: bytes,
		}
		root.Signatures = append(root.Signatures, sig)
	}
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
		fmt.Println("ERROR: Backup file exists:", path)
		fmt.Println("This file may be from a previous failed key rotation and include critical data. Please move this file somewhere safe before re-running this command.")
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

func findPrivKey(pubkey string, creds OfflineCreds) (*rsa.PrivateKey, error) {
	pubkey = strings.TrimSpace(pubkey)
	for k, v := range creds {
		if strings.HasSuffix(k, ".pub") {
			tk := client.AtsKey{}
			subcommands.DieNotNil(json.Unmarshal(v, &tk))
			if strings.TrimSpace(tk.KeyValue.Public) == pubkey {
				pkbytes := creds[strings.Replace(k, ".pub", ".sec", 1)]
				tk = client.AtsKey{}
				subcommands.DieNotNil(json.Unmarshal(pkbytes, &tk))
				privPem, _ := pem.Decode([]byte(tk.KeyValue.Private))
				if privPem == nil {
					return nil, fmt.Errorf("Unable to parse private key: %s", string(creds[k]))
				}
				if privPem.Type != "RSA PRIVATE KEY" {
					return nil, fmt.Errorf("Invalid private key???: %s", string(k))
				}
				pk, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
				return pk, err
			}
		}
	}
	return nil, fmt.Errorf("Can not find private key for: %s", pubkey)
}

func findRoot(root client.AtsTufRoot, creds OfflineCreds) (string, *rsa.PrivateKey, error) {
	kid := root.Signed.Roles["root"].KeyIDs[0]
	pub := root.Signed.Keys[kid].KeyValue.Public
	key, err := findPrivKey(pub, creds)
	return kid, key, err
}

func getOfflineCreds(credsFile string) OfflineCreds {
	f, err := os.Open(credsFile)
	subcommands.DieNotNil(err)
	defer f.Close()

	files := make(OfflineCreds)

	gzf, err := gzip.NewReader(f)
	subcommands.DieNotNil(err)
	tr := tar.NewReader(gzf)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		subcommands.DieNotNil(err)

		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		var b bytes.Buffer
		_, err = io.Copy(&b, tr)
		subcommands.DieNotNil(err)
		files[hdr.Name] = b.Bytes()
	}
	return files
}
