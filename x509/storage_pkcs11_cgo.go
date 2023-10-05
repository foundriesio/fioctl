//go:build !bashpki && cgopki

package x509

import (
	"crypto"
	"crypto/elliptic"
	"fmt"

	"github.com/ThalesIgnite/crypto11"

	"github.com/foundriesio/fioctl/subcommands"
)

const hsmObjectId = "1"

func newPkcs11Session(hsm HsmInfo) *crypto11.Context {
	cfg := crypto11.Config{
		Path:        hsm.Module,
		TokenLabel:  hsm.TokenLabel,
		Pin:         hsm.Pin,
		MaxSessions: 0,
	}

	ctx, err := crypto11.Configure(&cfg)
	subcommands.DieNotNil(err)
	return ctx
}

func genAndSaveKeyToHsm(hsm HsmInfo, id, label string) crypto.Signer {
	// See storage_pkcs11_tool.go why we need to first check for the key existence.
	ctx := newPkcs11Session(hsm)
	key, err := ctx.FindKeyPair([]byte(id), []byte(label))
	subcommands.DieNotNil(err)
	if key != nil {
		subcommands.DieNotNil(fmt.Errorf("Key %s already exists on the HSM device", label))
	}

	key, err = ctx.GenerateECDSAKeyPairWithLabel([]byte(id), []byte(label), elliptic.P256())
	subcommands.DieNotNil(err)
	return key
}

func loadKeyFromHsm(hsm HsmInfo, id, label string) crypto.Signer {
	ctx := newPkcs11Session(hsm)
	key, err := ctx.FindKeyPair([]byte(id), []byte(label))
	subcommands.DieNotNil(err)
	if key == nil {
		subcommands.DieNotNil(fmt.Errorf("Key %s not found on the HSM device", label))
	}
	return key
}

func (s *hsmStorage) genAndSaveKey() crypto.Signer {
	return genAndSaveKeyToHsm(s.HsmInfo, hsmObjectId, s.Label)
}

func (s *hsmStorage) loadKey() crypto.Signer {
	return loadKeyFromHsm(s.HsmInfo, hsmObjectId, s.Label)
}
