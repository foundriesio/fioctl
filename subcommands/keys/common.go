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

	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
)

type OfflineCreds map[string][]byte

type TufSigner struct {
	Id  string
	Key *rsa.PrivateKey
}

func SignMeta(metaBytes []byte, signers ...TufSigner) ([]tuf.Signature, error) {
	opts := rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}
	hashed := sha256.Sum256(metaBytes)

	signatures := make([]tuf.Signature, len(signers))

	for idx, signer := range signers {
		sigBytes, err := signer.Key.Sign(rand.Reader, hashed[:], &opts)
		if err != nil {
			return nil, err
		}
		signatures[idx] = tuf.Signature{
			KeyID:     signer.Id,
			Method:    "rsassa-pss-sha256",
			Signature: sigBytes,
		}
	}
	return signatures, nil
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

func FindPrivKey(pubkey string, creds OfflineCreds) (*rsa.PrivateKey, error) {
	pubkey = strings.TrimSpace(pubkey)
	for k, v := range creds {
		if strings.HasSuffix(k, ".pub") {
			tk := client.AtsKey{}
			if err := json.Unmarshal(v, &tk); err != nil {
				return nil, err
			}
			if strings.TrimSpace(tk.KeyValue.Public) == pubkey {
				pkbytes := creds[strings.Replace(k, ".pub", ".sec", 1)]
				tk = client.AtsKey{}
				if err := json.Unmarshal(pkbytes, &tk); err != nil {
					return nil, err
				}
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
