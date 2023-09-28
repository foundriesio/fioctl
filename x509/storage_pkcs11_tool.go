//go:build !bashpki

package x509

import (
	"crypto"
)

// TODO: The implementation is added in the next commit

func (s *hsmStorage) genAndSaveKey() crypto.Signer {
	panic("Not implemented")
}

func (s *hsmStorage) loadKey() crypto.Signer {
	panic("Not implemented")
}
