package keys

import (
	"crypto"
	"crypto/ed25519"
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
type tufKeyTypeEd25519 struct{}

const (
	// These are case insensitive
	tufRoleNameRoot = "Root"
	// tufRoleNameTimestamp  = "Timestamp"
	// tufRoleNameSnapshot   = "Snapshot"
	tufRoleNameTargets       = "Targets"
	tufKeyTypeNameEd25519    = "ED25519"
	tufKeyTypeNameRSA        = "RSA"
	tufKeyTypeSigNameEd25519 = "ed25519"
	tufKeyTypeSigNameRSA     = "rsassa-pss-sha256"
)

func parseTufRoleName(s string, supported ...string) (string, error) {
	su := strings.ToUpper(s)
	for _, ss := range supported {
		if su == strings.ToUpper(ss) {
			return ss, nil
		}
	}
	return "", fmt.Errorf("Unsupported role type: %s", s)
}

func parseTufKeyType(s string) (TufKeyType, error) {
	su := strings.ToUpper(s)
	switch su {
	case tufKeyTypeNameEd25519:
		return &tufKeyTypeEd25519{}, nil
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

func (t *tufKeyTypeEd25519) Name() string { return tufKeyTypeNameEd25519 }

func (t *tufKeyTypeEd25519) SigName() string { return tufKeyTypeSigNameEd25519 }

func (t *tufKeyTypeEd25519) SigOpts() crypto.SignerOpts {
	return crypto.Hash(0)
}

func (t *tufKeyTypeEd25519) GenerateKey() (crypto.Signer, error) {
	_, pk, err := ed25519.GenerateKey(rand.Reader)
	return pk, err
}

func (t *tufKeyTypeEd25519) ParseKey(priv string) (crypto.Signer, error) {
	pk, err := hex.DecodeString(priv)
	if err != nil {
		return nil, errors.New("Unable to parse Ed25519 private key HEX data")
	}
	switch len(pk) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(pk), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(pk), nil
	default:
		return nil, errors.New("Wrong Ed25519 private key size")
	}
}

func (t *tufKeyTypeEd25519) SaveKeyPair(key crypto.Signer) (priv, pub string, err error) {
	priv = hex.EncodeToString(key.(ed25519.PrivateKey).Seed())
	pub = hex.EncodeToString([]byte(key.Public().(ed25519.PublicKey)))
	return
}
