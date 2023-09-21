//go:build windows && (!dynamic || !hsm)

package x509

import (
	"errors"

	"github.com/foundriesio/fioctl/subcommands"
)

type KeyStorageHsm struct {
	Hsm HsmInfo
}

func (s *KeyStorageHsm) genAndSaveFactoryCaKey() (any, any) {
	dieHsmUnsupported()
	return nil, nil
}

func (s *KeyStorageHsm) getFactoryCaKey() any {
	dieHsmUnsupported()
	return nil
}

func dieHsmUnsupported() {
	subcommands.DieNotNil(errors.New("This Fioctl binary was built without HSM support"))
}
