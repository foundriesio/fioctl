package keys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	canonical "github.com/docker/go/canonical/json"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

type OfflineCreds map[string][]byte

type TufSigner struct {
	Id   string
	Type TufKeyType
	Key  crypto.Signer
}

type TufKeyPair struct {
	signer       TufSigner
	atsPriv      client.AtsKey
	atsPrivBytes []byte
	atsPub       client.AtsKey
	atsPubBytes  []byte
}

func ParseTufKeyType(s string) TufKeyType {
	t, err := parseTufKeyType(s)
	subcommands.DieNotNil(err)
	return t
}

func ParseTufRoleNameOffline(s string) string {
	r, err := parseTufRoleName(s, tufRoleNameRoot, tufRoleNameTargets)
	subcommands.DieNotNil(err)
	return r
}

func genTufKeyId(key crypto.Signer) string {
	// # This has to match the exact logic used by ota-tuf (required by garage-sign):
	// https://github.com/foundriesio/ota-tuf/blob/fio-changes/libtuf/src/main/scala/com/advancedtelematic/libtuf/crypt/TufCrypto.scala#L66-L71
	// It sets a keyid to a signature of the key's canonical DER encoding (same logic for all keys).
	// Note: this differs from the TUF spec, need to change once we deprecate the garage-sign.
	pubBytes, err := x509.MarshalPKIXPublicKey(key.Public())
	subcommands.DieNotNil(err)
	return fmt.Sprintf("%x", sha256.Sum256(pubBytes))
}

func genTufKeyPair(keyType TufKeyType) TufKeyPair {
	keyTypeName := keyType.Name()
	pk, err := keyType.GenerateKey()
	subcommands.DieNotNil(err)
	privKey, pubKey, err := keyType.SaveKeyPair(pk)
	subcommands.DieNotNil(err)

	priv := client.AtsKey{
		KeyType:  keyTypeName,
		KeyValue: client.AtsKeyVal{Private: privKey},
	}
	atsPrivBytes, err := json.Marshal(priv)
	subcommands.DieNotNil(err)

	pub := client.AtsKey{
		KeyType:  keyTypeName,
		KeyValue: client.AtsKeyVal{Public: pubKey},
	}
	atsPubBytes, err := json.Marshal(pub)
	subcommands.DieNotNil(err)

	id := genTufKeyId(pk)

	return TufKeyPair{
		atsPriv:      priv,
		atsPrivBytes: atsPrivBytes,
		atsPub:       pub,
		atsPubBytes:  atsPubBytes,
		signer: TufSigner{
			Id:   id,
			Type: keyType,
			Key:  pk,
		},
	}
}

func SignTufMeta(metaBytes []byte, signers ...TufSigner) ([]tuf.Signature, error) {
	signatures := make([]tuf.Signature, len(signers))

	for idx, signer := range signers {
		digest := metaBytes[:]
		opts := signer.Type.SigOpts()
		if opts.HashFunc() != crypto.Hash(0) {
			// Golang expects the caller to hash the digest if needed by the signing method

			h := opts.HashFunc().New()
			h.Write(digest)
			digest = h.Sum(nil)
		}
		sigBytes, err := signer.Key.Sign(rand.Reader, digest, opts)
		if err != nil {
			return nil, err
		}
		signatures[idx] = tuf.Signature{
			KeyID:     signer.Id,
			Method:    tuf.SigAlgorithm(signer.Type.SigName()),
			Signature: sigBytes,
		}
	}
	return signatures, nil
}

func signTufRoot(root *client.AtsTufRoot, signers ...TufSigner) error {
	bytes, err := canonical.MarshalCanonical(root.Signed)
	if err != nil {
		return err
	}
	signatures, err := SignTufMeta(bytes, signers...)
	if err != nil {
		return err
	}
	root.Signatures = signatures
	return nil
}

func saveTufCreds(path string, creds OfflineCreds) {
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
}

func saveTempTufCreds(credsFile string, creds OfflineCreds) string {
	path := credsFile + ".tmp"
	if _, err := os.Stat(path); err == nil {
		subcommands.DieNotNil(fmt.Errorf(`Backup file exists: %s
This file may be from a previous failed key rotation and include critical data.
Please move this file somewhere safe before re-running this command.`,
			path,
		))
	}
	saveTufCreds(path, creds)
	return path
}

func GetOfflineCreds(credsFile string) (OfflineCreds, error) {
	f, err := os.Open(credsFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	files := make(OfflineCreds)

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gzf)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return nil, err
		}

		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		var b bytes.Buffer
		if _, err = io.Copy(&b, tr); err != nil {
			return nil, err
		}
		files[hdr.Name] = b.Bytes()
	}
	return files, nil
}

func FindTufSigner(keyid, pubkey string, creds OfflineCreds) (*TufSigner, error) {
	pubkey = strings.TrimSpace(pubkey)
	for k, v := range creds {
		if strings.HasSuffix(k, ".pub") {
			tk := client.AtsKey{}
			if err := json.Unmarshal(v, &tk); err != nil {
				return nil, fmt.Errorf("Unable to parse JSON for %s: %w", k, err)
			}
			if strings.TrimSpace(tk.KeyValue.Public) == pubkey {
				pkname := strings.Replace(k, ".pub", ".sec", 1)
				pkbytes := creds[pkname]
				tk = client.AtsKey{}
				if err := json.Unmarshal(pkbytes, &tk); err != nil {
					return nil, fmt.Errorf("Unable to parse JSON for %s: %w", pkname, err)
				}
				keyType, err := parseTufKeyType(tk.KeyType)
				if err != nil {
					return nil, fmt.Errorf("Unsupported key type for %s: %s", pkname, tk.KeyType)
				}
				pk, err := keyType.ParseKey(tk.KeyValue.Private)
				if err != nil {
					return nil, fmt.Errorf("Unable to parse key value for %s: %w", pkname, err)
				}
				return &TufSigner{
					Id:   keyid,
					Type: keyType,
					Key:  pk,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("Can not find private key for: %s", keyid)
}

func findTufRootSigner(root *client.AtsTufRoot, creds OfflineCreds) (*TufSigner, error) {
	kid := root.Signed.Roles["root"].KeyIDs[0]
	pub := root.Signed.Keys[kid].KeyValue.Public
	return FindTufSigner(kid, pub, creds)
}

func removeUnusedTufKeys(root *client.AtsTufRoot) {
	var inuse []string
	for _, role := range root.Signed.Roles {
		inuse = append(inuse, role.KeyIDs...)
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

func checkTufRootUpdatesStatus(updates client.TufRootUpdates, forUpdate bool) (
	curCiRoot, newCiRoot *client.AtsTufRoot,
) {
	switch updates.Status {
	case client.TufRootUpdatesStatusNone:
		subcommands.DieNotNil(errors.New(`There are no TUF root updates in progress.
Please, run "fioctl keys tuf updates init" to start over.`))
	case client.TufRootUpdatesStatusStarted:
		break
	case client.TufRootUpdatesStatusApplying:
		if forUpdate {
			subcommands.DieNotNil(errors.New(
				"No modifications to TUF root updates allowed why they are being applied.",
			))
		}
	default:
		subcommands.DieNotNil(fmt.Errorf("Unexpected TUF root updates status: %s", updates.Status))
	}

	if updates.Current != nil && updates.Current.CiRoot != "" {
		subcommands.DieNotNil(
			json.Unmarshal([]byte(updates.Current.CiRoot), &curCiRoot), "Current CI root",
		)
	}
	if curCiRoot == nil {
		subcommands.DieNotNil(errors.New("Current TUF CI root not set. Please, report a bug."))
	}
	if updates.Updated != nil && updates.Updated.CiRoot != "" {
		subcommands.DieNotNil(
			json.Unmarshal([]byte(updates.Updated.CiRoot), &newCiRoot), "Updated CI root",
		)
	}
	if newCiRoot == nil {
		subcommands.DieNotNil(errors.New("Updated TUF CI root not set. Please, report a bug."))
	}
	return
}
