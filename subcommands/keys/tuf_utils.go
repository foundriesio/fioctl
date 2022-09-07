package keys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"encoding/json"
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

func GenKeyPair(keyType TufKeyType) TufKeyPair {
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

	id, err := pub.KeyID()
	subcommands.DieNotNil(err)

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

func SignMeta(metaBytes []byte, signers ...TufSigner) ([]tuf.Signature, error) {
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

func SignRoot(root *client.AtsTufRoot, signers ...TufSigner) error {
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

func SaveCreds(path string, creds OfflineCreds) {
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

func SaveTempCreds(credsFile string, creds OfflineCreds) string {
	path := credsFile + ".tmp"
	if _, err := os.Stat(path); err == nil {
		subcommands.DieNotNil(fmt.Errorf(`Backup file exists: %s
This file may be from a previous failed key rotation and include critical data.
Please move this file somewhere safe before re-running this command.`,
			path,
		))
	}
	SaveCreds(path, creds)
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

func FindSigner(keyid, pubkey string, creds OfflineCreds) (*TufSigner, error) {
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
	return nil, fmt.Errorf("Can not find private key for: %s", pubkey)
}

func RemoveUnusedKeys(root *client.AtsTufRoot) {
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
