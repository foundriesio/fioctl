package x509

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFactory    = "factory"
	testUser       = "fio-user"
	testDnsBase    = "ota-lite.fio"
	testDnsGateway = "repo.ota-lite.fio"
	testDnsOstree  = "repo.ostree.fio"
)

func TestNoHsm(t *testing.T) {
	runTest(t, func(factoryCa, tlsCert, onlineCa, offlineCa *x509.Certificate) {
		for _, fn := range []string{
			FactoryCaCertFile,
			FactoryCaKeyFile,
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

		factoryCaOnDisk, err := x509.ParseCertificate(pemToDer(t, readFile(FactoryCaCertFile)))
		require.Nil(t, err)
		factoryCaKeyOnDisk, err := x509.ParseECPrivateKey(pemToDer(t, readFile(FactoryCaKeyFile)))
		require.Nil(t, err)
		assert.Equal(t, factoryCa, factoryCaOnDisk)
		assert.Equal(t, factoryCa.PublicKey, factoryCaKeyOnDisk.Public())

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

func runTest(t *testing.T, verifyFiles func(factoryCa, tlsCert, onlineCa, offlineCa *x509.Certificate)) {
	dir, err := os.MkdirTemp("", "test-certs-*")
	require.Nil(t, err)
	defer os.RemoveAll(dir)
	require.Nil(t, os.Chdir(dir))

	factoryCaPool := x509.NewCertPool()
	tlsKey, tlsCsr := genTestTlsCsr(t)
	onlineCaKey, onlineCaCsr := genTestOnlineCaCsr(t)
	el2goCaKey, el2goCaCsr := genTestEl2GoCaCsr(t)

	factoryCaPem := CreateFactoryCa(testFactory)
	factoryCa, err := x509.ParseCertificate(pemToDer(t, factoryCaPem))
	require.Nil(t, err)
	factoryCaPool.AddCert(factoryCa)
	factoryCaChain, err := factoryCa.Verify(x509.VerifyOptions{
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	assert.Nil(t, err)

	assert.Equal(t, true, factoryCa.IsCA)
	assert.Equal(t, x509.KeyUsageCertSign, factoryCa.KeyUsage)
	assert.Equal(t, factoryCaName, factoryCa.Subject.CommonName)
	assert.Equal(t, []string{testFactory}, factoryCa.Subject.OrganizationalUnit)
	assert.Equal(t, 1, len(factoryCaChain))
	assert.Equal(t, 1, len(factoryCaChain[0]))
	assert.Equal(t, factoryCa, factoryCaChain[0][0])

	tlsCertPem := SignTlsCsr(tlsCsr)
	tlsCert, err := x509.ParseCertificate(pemToDer(t, tlsCertPem))
	require.Nil(t, err)
	tlsCertChain, err := tlsCert.Verify(x509.VerifyOptions{
		DNSName:   testDnsGateway,
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	assert.Nil(t, err)
	tlsCertChain1, err := tlsCert.Verify(x509.VerifyOptions{
		DNSName:   testDnsOstree,
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	assert.Nil(t, err)
	assert.Equal(t, tlsCertChain, tlsCertChain1)

	assert.Equal(t, false, tlsCert.IsCA)
	assert.Equal(t, x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment|x509.KeyUsageKeyAgreement, tlsCert.KeyUsage)
	assert.Equal(t, testDnsBase, tlsCert.Subject.CommonName)
	assert.Equal(t, 2, len(tlsCert.DNSNames))
	assert.Equal(t, testDnsGateway, tlsCert.DNSNames[0])
	assert.Equal(t, testDnsOstree, tlsCert.DNSNames[1])
	assert.Equal(t, 1, len(tlsCertChain))
	assert.Equal(t, 2, len(tlsCertChain[0]))
	assert.Equal(t, tlsCert.PublicKey, tlsKey.Public())
	assert.Equal(t, tlsCert, tlsCertChain[0][0])
	assert.Equal(t, factoryCa, tlsCertChain[0][1])

	onlineCaPem := SignCaCsr(onlineCaCsr)
	onlineCa, err := x509.ParseCertificate(pemToDer(t, onlineCaPem))
	require.Nil(t, err)
	onlineCaChain, err := onlineCa.Verify(x509.VerifyOptions{
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{},
	})
	assert.Nil(t, err)

	assert.Equal(t, true, onlineCa.IsCA)
	assert.Equal(t, x509.KeyUsageCertSign, onlineCa.KeyUsage)
	assert.Equal(t, testDnsGateway, onlineCa.Subject.CommonName)
	assert.Equal(t, []string{testFactory}, onlineCa.Subject.OrganizationalUnit)
	assert.Equal(t, 1, len(onlineCaChain))
	assert.Equal(t, 2, len(onlineCaChain[0]))
	assert.Equal(t, onlineCa.PublicKey, onlineCaKey.Public())
	assert.Equal(t, onlineCa, onlineCaChain[0][0])
	assert.Equal(t, factoryCa, onlineCaChain[0][1])

	offlineCaPem := CreateDeviceCa(testUser, testFactory)
	offlineCa, err := x509.ParseCertificate(pemToDer(t, offlineCaPem))
	require.Nil(t, err)
	offlineCaChain, err := offlineCa.Verify(x509.VerifyOptions{
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{},
	})
	assert.Nil(t, err)

	assert.Equal(t, true, offlineCa.IsCA)
	assert.Equal(t, x509.KeyUsageCertSign, offlineCa.KeyUsage)
	assert.Equal(t, testUser, offlineCa.Subject.CommonName)
	assert.Equal(t, []string{testFactory}, offlineCa.Subject.OrganizationalUnit)
	assert.Equal(t, 1, len(offlineCaChain))
	assert.Equal(t, 2, len(offlineCaChain[0]))
	assert.Equal(t, offlineCa, offlineCaChain[0][0])
	assert.Equal(t, factoryCa, offlineCaChain[0][1])

	el2goCaPem := SignEl2GoCsr(el2goCaCsr)
	el2goCa, err := x509.ParseCertificate(pemToDer(t, el2goCaPem))
	require.Nil(t, err)
	el2goCaChain, err := el2goCa.Verify(x509.VerifyOptions{
		Roots:     factoryCaPool,
		KeyUsages: []x509.ExtKeyUsage{},
	})
	assert.Nil(t, err)

	assert.Equal(t, true, el2goCa.IsCA)
	assert.Equal(t, x509.KeyUsageCertSign, el2goCa.KeyUsage)
	assert.Equal(t, testDnsGateway, el2goCa.Subject.CommonName)
	assert.Equal(t, []string{testFactory}, el2goCa.Subject.OrganizationalUnit)
	assert.Equal(t, []string{"nxp"}, el2goCa.Subject.Organization)
	assert.Equal(t, 1, len(el2goCaChain))
	assert.Equal(t, 2, len(el2goCaChain[0]))
	assert.Equal(t, el2goCa.PublicKey, el2goCaKey.Public())
	assert.Equal(t, el2goCa, el2goCaChain[0][0])
	assert.Equal(t, factoryCa, el2goCaChain[0][1])

	verifyFiles(factoryCa, tlsCert, onlineCa, offlineCa)
}

func pemToDer(t *testing.T, block string) []byte {
	der, rest := pem.Decode([]byte(block))
	require.NotNil(t, der)
	require.Equal(t, 0, len(rest))
	return der.Bytes
}

func genTestTlsCsr(t *testing.T) (crypto.Signer, string) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)
	csr := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: testDnsBase},
		DNSNames: []string{testDnsGateway, testDnsOstree},
	}
	csrDer, err := x509.CreateCertificateRequest(rand.Reader, csr, key)
	require.Nil(t, err)
	csrPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDer})
	return key, string(csrPem)
}

func genTestOnlineCaCsr(t *testing.T) (crypto.Signer, string) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)
	csr := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: testDnsGateway, OrganizationalUnit: []string{testFactory}},
	}
	csrDer, err := x509.CreateCertificateRequest(rand.Reader, csr, key)
	require.Nil(t, err)
	csrPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDer})
	return key, string(csrPem)
}

func genTestEl2GoCaCsr(t *testing.T) (crypto.Signer, string) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)
	csr := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         testDnsGateway,
			OrganizationalUnit: []string{testFactory},
			Organization:       []string{"nxp"},
		},
	}
	csrDer, err := x509.CreateCertificateRequest(rand.Reader, csr, key)
	require.Nil(t, err)
	csrPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDer})
	return key, string(csrPem)
}
