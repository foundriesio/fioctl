//go:build !bashpki

package x509

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"github.com/foundriesio/fioctl/subcommands"
)

type KeyStorage interface {
	genAndSaveKey() crypto.Signer
	loadKey() crypto.Signer
}

func genRandomSerialNumber() *big.Int {
	// Generate a 160 bits serial number (20 octets)
	max := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(160), nil)
	serial, err := rand.Int(rand.Reader, max)
	subcommands.DieNotNil(err)
	return serial
}

func genCertificate(
	crtTemplate *x509.Certificate, caCrt *x509.Certificate, pub crypto.PublicKey, signerKey crypto.Signer,
) string {
	certRaw, err := x509.CreateCertificate(rand.Reader, crtTemplate, caCrt, pub, signerKey)
	subcommands.DieNotNil(err)

	certPemBlock := pem.Block{Type: "CERTIFICATE", Bytes: certRaw}
	var certRow bytes.Buffer
	err = pem.Encode(&certRow, &certPemBlock)
	subcommands.DieNotNil(err)

	return certRow.String()
}

func parseOnePemBlock(pemBlock string) *pem.Block {
	first, rest := pem.Decode([]byte(pemBlock))
	if first == nil || len(rest) > 0 {
		subcommands.DieNotNil(errors.New("Malformed PEM data"))
	}
	return first
}

func parsePemCertificateRequest(csrPem string) *x509.CertificateRequest {
	pemBlock := parseOnePemBlock(csrPem)
	clientCSR, err := x509.ParseCertificateRequest(pemBlock.Bytes)
	subcommands.DieNotNil(err)
	err = clientCSR.CheckSignature()
	subcommands.DieNotNil(err)
	return clientCSR
}

func marshalSubject(cn string, ou string) pkix.Name {
	// In it's simpler form, this function would be replaced by
	// pkix.Name{CommonName: cn, OrganizationalUnit: []string{ou}}
	// However, x509 library uses PrintableString instead of UTF8String
	// as ASN.1 field type. This function forces UTF8String instead, to
	// avoid compatibility issues when using a device certificate created
	// with libraries such as MbedTLS.
	// x509 library also encodes OU and CN in a different order if compared
	// to OpenSSL, which is less of an issue, but still worth to adjust
	// while we are at it.
	cnBytes, err := asn1.MarshalWithParams(cn, "utf8")
	subcommands.DieNotNil(err)
	ouBytes, err := asn1.MarshalWithParams(ou, "utf8")
	subcommands.DieNotNil(err)
	var (
		oidCommonName         = []int{2, 5, 4, 3}
		oidOrganizationalUnit = []int{2, 5, 4, 11}
	)
	pkixAttrTypeValue := []pkix.AttributeTypeAndValue{
		{
			Type:  oidCommonName,
			Value: asn1.RawValue{FullBytes: cnBytes},
		},
		{
			Type:  oidOrganizationalUnit,
			Value: asn1.RawValue{FullBytes: ouBytes},
		},
	}
	return pkix.Name{ExtraNames: pkixAttrTypeValue}
}

func CreateFactoryCa(ou string) string {
	priv := factoryCaKeyStorage.genAndSaveKey()
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      marshalSubject(factoryCaName, ou),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(20, 0, 0),

		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	factoryCaString := genCertificate(&crtTemplate, &crtTemplate, priv.Public(), priv)
	writeFile(FactoryCaCertFile, factoryCaString)
	return factoryCaString
}

func CreateDeviceCa(cn string, ou string) string {
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := loadCertFromFile(FactoryCaCertFile)
	priv := genAndSaveKeyToFile(DeviceCaKeyFile)
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      marshalSubject(cn, ou),
		Issuer:       factoryCa.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	crtPem := genCertificate(&crtTemplate, factoryCa, priv.Public(), factoryKey)
	writeFile(DeviceCaCertFile, crtPem)
	return crtPem
}

func SignTlsCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := loadCertFromFile(FactoryCaCertFile)
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      csr.Subject,
		Issuer:       factoryCa.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		IsCA:        false,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageKeyAgreement,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    csr.DNSNames,
	}
	crtPem := genCertificate(&crtTemplate, factoryCa, csr.PublicKey, factoryKey)
	writeFile(TlsCertFile, crtPem)
	return crtPem
}

func SignCaCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := loadCertFromFile(FactoryCaCertFile)
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      csr.Subject,
		Issuer:       factoryCa.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	crtPem := genCertificate(&crtTemplate, factoryCa, csr.PublicKey, factoryKey)
	writeFile(OnlineCaCertFile, crtPem)
	return crtPem
}

func SignEl2GoCsr(csrPem string) string {
	return SignCaCsr(csrPem)
}
