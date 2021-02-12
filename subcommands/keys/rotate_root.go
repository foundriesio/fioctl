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
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	canonical "github.com/docker/go/canonical/json"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	subcommands.DieNotNil(syncProdRoot(factory, *root, creds))

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

	bytes, err := json.MarshalIndent(root, "", "  ")
	subcommands.DieNotNil(err)

	// Create a backup before we try and commit this:
	tmpCreds := saveTempCreds(credsFile, newCreds)

	fmt.Println("= Uploading rotated root")
	body, err := api.TufRootPost(factory, bytes)
	if err != nil {
		fmt.Println("\nERROR:", err)
		fmt.Println(body)
		os.Exit(1)
	}
	if err := os.Rename(tmpCreds, credsFile); err != nil {
		fmt.Println("\nERROR: Unable to update offline creds file.", err)
		fmt.Println("Temp copy still available at:", tmpCreds)
	}

	// backfill this new key
	subcommands.DieNotNil(syncProdRoot(factory, *root, creds))
}

func removeUnusedKeys(root *client.TufRoot) {
	var inuse []string
	for _, role := range root.Signed.Roles {
		inuse = append(inuse, role.KeyIDs...)
	}
	// we also have to be careful to not loose the extra root key when doing
	// a root key rotation
	for _, sig := range root.Signatures {
		inuse = append(inuse, sig.KeyId)
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

func signRoot(root *client.TufRoot, signers ...TufSigner) error {
	bytes, err := canonical.MarshalCanonical(root.Signed)
	if err != nil {
		return err
	}

	opts := rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}
	hashed := sha256.Sum256(bytes)

	root.Signatures = []client.TufSig{}

	for _, signer := range signers {
		bytes, err = signer.Key.Sign(rand.Reader, hashed[:], &opts)
		if err != nil {
			return err
		}
		sig := client.TufSig{
			KeyId:     signer.Id,
			Method:    "rsassa-pss-sha256",
			Signature: base64.StdEncoding.EncodeToString(bytes),
		}
		root.Signatures = append(root.Signatures, sig)
	}
	return nil
}

func swapRootKey(root *client.TufRoot, curid string, creds OfflineCreds) (string, *rsa.PrivateKey, OfflineCreds) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	subcommands.DieNotNil(err)

	var privBytes []byte = x509.MarshalPKCS1PrivateKey(pk)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privBytes, err = json.Marshal(client.TufKey{
		KeyType:  "RSA",
		KeyValue: client.TufKeyVal{Private: string(pem.EncodeToMemory(block))},
	})
	subcommands.DieNotNil(err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	subcommands.DieNotNil(err)

	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	id := fmt.Sprintf("%x", sha256.Sum256(privBytes))
	root.Signed.Keys[id] = client.TufKey{
		KeyType:  "RSA",
		KeyValue: client.TufKeyVal{Public: string(pem.EncodeToMemory(block))},
	}
	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Roles["root"].KeyIDs = []string{id}
	root.Signed.Version += 1

	pubBytes, err = json.Marshal(root.Signed.Keys[id])
	subcommands.DieNotNil(err)

	base := "tufrepo/keys/fioctl-root-" + id
	creds[base+".pub"] = pubBytes
	creds[base+".sec"] = privBytes
	return id, pk, creds
}

func assertWritable(path string) {
	err := syscall.Access(path, syscall.O_RDWR)
	if err != nil {
		fmt.Println("ERROR: File is not writeable:", path)
		os.Exit(1)
	}
}

func saveTempCreds(credsFile string, creds OfflineCreds) string {
	path := credsFile + ".tmp"
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
			tk := client.TufKey{}
			subcommands.DieNotNil(json.Unmarshal(v, &tk))
			if strings.TrimSpace(tk.KeyValue.Public) == pubkey {
				pkbytes := creds[strings.Replace(k, ".pub", ".sec", 1)]
				tk = client.TufKey{}
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

func findRoot(root client.TufRoot, creds OfflineCreds) (string, *rsa.PrivateKey, error) {
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

func syncProdRoot(factory string, curRoot client.TufRoot, creds OfflineCreds) error {
	curProd, err := api.TufProdRootGet(factory)
	if err != nil {
		if httpE := client.AsHttpError(err); httpE != nil {
			if httpE.Response.StatusCode == 404 {
				// this is okay. it means we need to create the 1.rootjson
				err = nil
			}
		}
	}
	if err != nil {
		return err
	}
	prodVer := 0
	if curProd != nil {
		prodVer = curProd.Signed.Version
	}
	if prodVer == curRoot.Signed.Version {
		return nil
	}

	for i := prodVer + 1; i <= curRoot.Signed.Version; i++ {
		fmt.Println("= Populating production root version", i)
		root, err := api.TufRootGetVer(factory, i)
		if err != nil {
			return err
		}

		// Bump the threshold
		root.Signed.Roles["targets"].Threshold = 2

		// Sign with the same keys used for the ci copy
		var signers []TufSigner
		for _, sig := range root.Signatures {
			key, ok := root.Signed.Keys[sig.KeyId]
			if !ok && i > 1 {
				// Root key was rotated, this pub key is previous version
				prev, err := api.TufRootGetVer(factory, i-1)
				if err != nil {
					return err
				}
				key = prev.Signed.Keys[sig.KeyId]
			}
			pkey, err := findPrivKey(key.KeyValue.Public, creds)
			if err != nil {
				return err
			}
			signers = append(signers, TufSigner{
				Id:  sig.KeyId,
				Key: pkey,
			})
		}
		if err := signRoot(root, signers...); err != nil {
			return err
		}

		bytes, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return err
		}
		body, err := api.TufProdRootPost(factory, bytes)
		if err != nil {
			return fmt.Errorf("%s: %s", err, body)
		}
	}
	return nil
}
