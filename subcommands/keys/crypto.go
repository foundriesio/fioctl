package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

type TufKeyType interface {
	Name() string
	SigName() string
	SigOpts() crypto.SignerOpts
	GenerateKey() (crypto.Signer, error)
	ParseKey(string) (crypto.Signer, error)
	SaveKeyPair(crypto.Signer) (priv, pub string, err error)
}

type tufKeyTypeRSA struct{}

const (
	// These are case insensitive
	tufKeyTypeNameRSA    = "RSA"
	tufKeyTypeSigNameRSA = "rsassa-pss-sha256"
)

func parseTufKeyType(s string) (TufKeyType, error) {
	su := strings.ToUpper(s)
	switch su {
	case tufKeyTypeNameRSA:
		return &tufKeyTypeRSA{}, nil
	default:
		return nil, fmt.Errorf("Unsupported key type: %s", s)
	}
}

func (t *tufKeyTypeRSA) Name() string { return tufKeyTypeNameRSA }

func (t *tufKeyTypeRSA) SigName() string { return tufKeyTypeSigNameRSA }

func (t *tufKeyTypeRSA) SigOpts() crypto.SignerOpts {
	return &rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}
}

func (t *tufKeyTypeRSA) GenerateKey() (crypto.Signer, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}

func (t *tufKeyTypeRSA) ParseKey(priv string) (crypto.Signer, error) {
	der, _ := pem.Decode([]byte(priv))
	if der == nil {
		return nil, errors.New("Unable to parse RSA private key PEM data")
	}
	pk, err := x509.ParsePKCS1PrivateKey(der.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse RSA private key PKCS1 DER data: %w", err)
	}
	return pk, nil
}

func (t *tufKeyTypeRSA) SaveKeyPair(key crypto.Signer) (priv, pub string, err error) {
	privBytes := x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))
	pubBytes, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return
	}
	priv = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}))
	pub = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}))
	return
}
