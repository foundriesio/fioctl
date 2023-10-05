//go:build !bashpki && !cgopki

package x509

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/foundriesio/fioctl/subcommands"
)

const hsmObjectId = "1"

type hsmSigner struct {
	hsm   HsmInfo
	id    string
	label string
	pub   crypto.PublicKey
}

func (s *hsmSigner) keyArgs() []string {
	return []string{
		"--module",
		s.hsm.Module,
		"--token-label",
		s.hsm.TokenLabel,
		"--pin",
		s.hsm.Pin,
		"--id",
		s.id,
		"--label",
		s.label,
	}
}

func (s *hsmSigner) Public() crypto.PublicKey {
	if s.pub != nil {
		return s.pub
	}
	args := append(s.keyArgs(), "--read-object", "--type=pubkey")
	cmd := exec.Command("pkcs11-tool", args...)
	out, err := cmd.Output()
	var ex *exec.ExitError
	if errors.As(err, &ex) {
		if strings.HasPrefix(string(ex.Stderr), "error: object not found") {
			return nil
		}
		fmt.Println(string(ex.Stderr))
	}
	subcommands.DieNotNil(err)
	key, err := x509.ParsePKIXPublicKey(out)
	subcommands.DieNotNil(err)
	s.pub = key
	return key
}

func (s *hsmSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// By default pkcs11-tool returns raw signature, X509 needs it wrapped into the ASN.1 sequence
	args := append(s.keyArgs(), "--sign", "--mechanism=ECDSA", "--signature-format=sequence")
	cmd := exec.Command("pkcs11-tool", args...)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	subcommands.DieNotNil(err)

	go func() {
		defer in.Close()
		_, err := in.Write(digest)
		subcommands.DieNotNil(err)
	}()

	return cmd.Output()
}

func genAndSaveKeyToHsm(hsm HsmInfo, id, label string) crypto.Signer {
	// The pkcs11-tool allows creating many objects with the same ID and label, potentially corrupting the storage.
	// For now, allow to create only one object.
	// In the future we will use the ID field to allow key rotation.
	key := &hsmSigner{hsm, id, label, nil}
	if key.Public() != nil {
		subcommands.DieNotNil(fmt.Errorf("Key %s already exists on the HSM device", label))
	}

	args := append(key.keyArgs(), "--keypairgen", "--key-type=EC:prime256v1")
	cmd := exec.Command("pkcs11-tool", args...)
	cmd.Stderr = os.Stderr
	subcommands.DieNotNil(cmd.Run())
	return key
}

func loadKeyFromHsm(hsm HsmInfo, id, label string) crypto.Signer {
	key := &hsmSigner{hsm, id, label, nil}
	if key.Public() == nil {
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
