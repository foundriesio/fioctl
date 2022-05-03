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
	"github.com/foundriesio/fioctl/subcommands"
)

type OfflineCreds map[string][]byte

type TufSigner struct {
	Id  string
	Key *rsa.PrivateKey
}

type TufKeyPair struct {
	rsaPriv      *rsa.PrivateKey
	atsPriv      client.AtsKey
	atsPrivBytes []byte

	atsPub      client.AtsKey
	atsPubBytes []byte

	keyid string
}

func GenKeyPair() TufKeyPair {
	pk, err := rsa.GenerateKey(rand.Reader, 4096)
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

	return TufKeyPair{
		atsPriv:      priv,
		atsPrivBytes: atsPrivBytes,
		atsPub:       pub,
		atsPubBytes:  atsPubBytes,
		keyid:        id,
		rsaPriv:      pk,
	}
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
