//go:build testhsm

package x509

import (
	"crypto/x509"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHsmModule     = "/usr/lib/softhsm/libsofthsm2.so"
	testHsmPin        = "1234"
	testHsmTokenLabel = "fioctl-test"
)

func TestHsm(t *testing.T) {
	softHsmTokenDir, err := os.MkdirTemp("", "softhsm-tokens-*")
	require.Nil(t, err)
	defer os.RemoveAll(softHsmTokenDir)

	softHsmConfig, err := os.CreateTemp("", "softhsm-config.cfg")
	require.Nil(t, err)
	defer os.Remove(softHsmConfig.Name())
	func() {
		defer softHsmConfig.Close()
		_, err := softHsmConfig.Write([]byte("directories.tokendir = " + softHsmTokenDir))
		require.Nil(t, err)
	}()
	os.Setenv("SOFTHSM2_CONF", softHsmConfig.Name())

	cmd := exec.Command(
		"softhsm2-util", "--init-token", "--slot", "0",
		"--label", testHsmTokenLabel, "--so-pin", testHsmPin, "--pin", testHsmPin)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	require.Nil(t, cmd.Run())

	InitHsm(HsmInfo{Module: testHsmModule, Pin: testHsmPin, TokenLabel: testHsmTokenLabel})

	runTest(t, func(factoryCa, tlsCert, onlineCa, offlineCa *x509.Certificate) {
		for _, fn := range []string{
			FactoryCaCertFile,
			TlsCertFile,
			OnlineCaCertFile,
			DeviceCaCertFile,
			DeviceCaKeyFile,
		} {
			stat, err := os.Lstat(fn)
			require.Nil(t, err)
			assert.Equal(t, fn, stat.Name())
			assert.Equal(t, false, stat.IsDir())
			assert.Equal(t, os.FileMode(0400), stat.Mode())
		}
		for _, fn := range []string{
			FactoryCaKeyFile,
		} {
			_, err := os.Lstat(fn)
			require.NotNil(t, err)
			require.Equal(t, true, os.IsNotExist(err))
		}

		factoryCaOnDisk, err := x509.ParseCertificate(pemToDer(t, readFile(FactoryCaCertFile)))
		require.Nil(t, err)
		factoryCaPubeyOnHsm, err := x509.ParsePKIXPublicKey(readPubkeyFromHsm(t, FactoryCaKeyLabel))
		require.Nil(t, err)
		assert.Equal(t, factoryCa, factoryCaOnDisk)
		assert.Equal(t, factoryCa.PublicKey, factoryCaPubeyOnHsm)

		tlsCertOnDisk, err := x509.ParseCertificate(pemToDer(t, readFile(TlsCertFile)))
		require.Nil(t, err)
		assert.Equal(t, tlsCert, tlsCertOnDisk)

		onlineCaOnDisk, err := x509.ParseCertificate(pemToDer(t, readFile(OnlineCaCertFile)))
		require.Nil(t, err)
		assert.Equal(t, onlineCa, onlineCaOnDisk)

		offlineCaOnDisk, err := x509.ParseCertificate(pemToDer(t, readFile(DeviceCaCertFile)))
		require.Nil(t, err)
		offlineCaKeyOnDisk, err := x509.ParseECPrivateKey(pemToDer(t, readFile(DeviceCaKeyFile)))
		require.Nil(t, err)
		assert.Equal(t, offlineCa, offlineCaOnDisk)
		assert.Equal(t, offlineCa.PublicKey, offlineCaKeyOnDisk.Public())
	})
}

func readPubkeyFromHsm(t *testing.T, label string) []byte {
	cmd := exec.Command(
		"pkcs11-tool", "--module", testHsmModule, "--read-object", "--type", "pubkey",
		"--token-label", testHsmTokenLabel, "--pin", testHsmPin, "--label", label)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	require.Nil(t, err)
	return out
}
