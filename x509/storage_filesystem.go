//go:build !bashpki

package x509

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/foundriesio/fioctl/subcommands"
)

func genAndSaveKeyToFile(fn string) crypto.Signer {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	subcommands.DieNotNil(err)

	keyRaw, err := x509.MarshalECPrivateKey(priv)
	subcommands.DieNotNil(err)

	keyBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyRaw}
	keyBytes := pem.EncodeToMemory(keyBlock)
	err = os.WriteFile(fn, keyBytes, 0600)
	subcommands.DieNotNil(err)
	return priv
}

func loadKeyFromFile(fn string) crypto.Signer {
	keyPem := parseOnePemBlock(readFile(fn))
	key, err := x509.ParseECPrivateKey(keyPem.Bytes)
	subcommands.DieNotNil(err)
	return key
}

func loadCertFromFile(fn string) *x509.Certificate {
	crtPem := parseOnePemBlock(readFile(fn))
	crt, err := x509.ParseCertificate(crtPem.Bytes)
	subcommands.DieNotNil(err)
	return crt
}
