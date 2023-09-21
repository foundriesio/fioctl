//go:build windows && dynamic && hsm

package x509

import (
	"crypto"
	"encoding/asn1"
	"errors"
	"fmt"
	"math"

	"github.com/ThalesIgnite/crypto11"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/miekg/pkcs11"
)

type KeyStorageHsm struct {
	Hsm HsmInfo
}

func (s *KeyStorageHsm) genAndSaveFactoryCaKey() (any, any) {
	return s.genAndSaveKeyToHsm([]byte(FactoryCaKeyPkcs11Id), []byte(FactoryCaKeyPkcs11Label))
}

func (s *KeyStorageHsm) getFactoryCaKey() any {
	return s.getKeyHsm([]byte(FactoryCaKeyPkcs11Id))
}

func (s *KeyStorageHsm) getKeyHsm(id []byte) crypto11.Signer {
	cfg := crypto11.Config{
		Path:        s.Hsm.Module,
		TokenLabel:  s.Hsm.TokenLabel,
		Pin:         s.Hsm.Pin,
		MaxSessions: 0,
	}

	ctx, err := crypto11.Configure(&cfg)
	subcommands.DieNotNil(err)

	privKey, err := ctx.FindKeyPair(id, nil)
	subcommands.DieNotNil(err)
	if privKey == nil {
		subcommands.DieNotNil(errors.New("Key with requested id not found"))
	}

	return privKey
}
func (s *KeyStorageHsm) genAndSaveKeyToHsm(id []byte, label []byte) (crypto11.Signer, crypto.PublicKey) {
	// Need all deferred statements to commit before the getKeyHsm call below.
	func() {
		p := pkcs11.New(s.Hsm.Module)
		err := p.Initialize()
		subcommands.DieNotNil(err)

		defer p.Destroy()
		defer p.Finalize()

		slots, err := p.GetSlotList(true)
		subcommands.DieNotNil(err)

		var matchedSlotId uint = math.MaxUint
		for _, slotId := range slots {
			token, err := p.GetTokenInfo(slotId)
			if err != nil {
				continue
			}
			if token.Label == s.Hsm.TokenLabel {
				matchedSlotId = slotId
				break
			}
		}

		if matchedSlotId == math.MaxUint {
			subcommands.DieNotNil(errors.New("PKCS#11: Unable to find Token" + s.Hsm.TokenLabel))
		} else {
			fmt.Printf("PKCS#11: Found Token %s at SlotID %d\n", s.Hsm.TokenLabel, matchedSlotId)
		}

		session, err := p.OpenSession(matchedSlotId, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
		subcommands.DieNotNil(err)
		defer p.CloseSession(session)

		err = p.Login(session, pkcs11.CKU_USER, s.Hsm.Pin)
		subcommands.DieNotNil(err)
		defer p.Logout(session)

		oidNamedCurveP256 := asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
		marshaledOID, err := asn1.Marshal(oidNamedCurveP256)

		publicKeyTemplate := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_EC),
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, marshaledOID),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
			pkcs11.NewAttribute(pkcs11.CKA_DERIVE, true),
		}
		privateKeyTemplate := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_EC),
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
			pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_DERIVE, true),
		}
		_, _, err = p.GenerateKeyPair(session,
			[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_EC_KEY_PAIR_GEN, nil)},
			publicKeyTemplate, privateKeyTemplate)
		subcommands.DieNotNil(err)
	}()
	priv := s.getKeyHsm(id)
	return priv, priv.Public()
}
