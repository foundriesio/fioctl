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
	"math/big"
	"time"

	"github.com/foundriesio/fioctl/subcommands"
)

type KeyStorage interface {
	genAndSaveKey() crypto.Signer
	loadKey() crypto.Signer
}

func CreateFactoryCa(ou string) string {
	priv := factoryCaKeyStorage.genAndSaveKey()
	crtTemplate := genFactoryCaTemplate(marshalSubject(factoryCaName, ou))
	factoryCaString := genCertificate(crtTemplate, crtTemplate, priv.Public(), priv)
	writeFile(FactoryCaCertFile, factoryCaString)
	return factoryCaString
}

func CreateFactoryCrossCa(ou string, pubkey crypto.PublicKey) string {
	// Cross-signed factory CA has all the same properties as a factory CA, but is signed by another factory CA.
	// This function does an inverse: produces a factory CA with a public key "borrowed" from another factory CA.
	// The end result is the same, but we don't need to export the internal key storage interface.
	// This certificate is not written to disk, as it is only needed intermittently.
	// Cannot use a ReSignCrt as we need a new certificate here (with e.g. a new serial number).
	priv := factoryCaKeyStorage.loadKey()
	crtTemplate := genFactoryCaTemplate(marshalSubject(factoryCaName, ou))
	return genCertificate(crtTemplate, crtTemplate, pubkey, priv)
}

func CreateDeviceCa(cn, ou string) string {
	return CreateDeviceCaExt(cn, ou, DeviceCaKeyFile, DeviceCaCertFile)
}

func CreateDeviceCaExt(cn, ou, keyFile, certFile string) string {
	priv := genAndSaveKeyToFile(keyFile)
	crtPem := genDeviceCaCert(marshalSubject(cn, ou), priv.Public())
	writeFile(certFile, crtPem)
	return crtPem
}

func SignCaCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	crtPem := genDeviceCaCert(csr.Subject, csr.PublicKey)
	writeFile(OnlineCaCertFile, crtPem)
	return crtPem
}

func SignEl2GoCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	return genDeviceCaCert(csr.Subject, csr.PublicKey)
}

func SignTlsCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	crtPem := genTlsCert(csr.Subject, csr.DNSNames, csr.PublicKey)
	writeFile(TlsCertFile, crtPem)
	return crtPem
}

func SignEstCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	return genTlsCert(csr.Subject, csr.DNSNames, csr.PublicKey)
}

func ReSignCrt(crt *x509.Certificate) string {
	// Use an input certificate as a template for a new certificate, preserving all its properties except a signature.
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := LoadCertFromFile(FactoryCaCertFile)
	return genCertificate(crt, factoryCa, crt.PublicKey, factoryKey)
}

var oidExtensionReasonCode = []int{2, 5, 29, 21}

func CreateCrl(serials map[string]int) string {
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := LoadCertFromFile(FactoryCaCertFile)
	now := time.Now()
	crl := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: now,
		NextUpdate: now.Add(time.Minute * 15),
	}
	for serial, reason := range serials {
		num := new(big.Int)
		if _, ok := num.SetString(serial, 10); !ok {
			// We expect a valid input here
			panic("Value is not a valid base 10 serial:" + serial)
		}
		// This would be easier with RevokedCertificateEntries, but it is not yet available in Golang 1.20.
		reasonBytes, err := asn1.Marshal(asn1.Enumerated(reason))
		subcommands.DieNotNil(err)

		crl.RevokedCertificates = append(crl.RevokedCertificates, pkix.RevokedCertificate{
			SerialNumber:   num,
			RevocationTime: now,
			Extensions:     []pkix.Extension{{Id: oidExtensionReasonCode, Value: reasonBytes}},
		})
	}
	// Our old root CAs are missing this bit; it's OK to add this here, and the API will accept it.
	factoryCa.KeyUsage |= x509.KeyUsageCRLSign
	derBytes, err := x509.CreateRevocationList(rand.Reader, crl, factoryCa, factoryKey)
	subcommands.DieNotNil(err)

	var pemBuffer bytes.Buffer
	err = pem.Encode(&pemBuffer, &pem.Block{Type: "X509 CRL", Bytes: derBytes})
	subcommands.DieNotNil(err)
	return pemBuffer.String()
}

func genFactoryCaTemplate(subject pkix.Name) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(20, 0, 0),

		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
}

func genTlsCert(subject pkix.Name, dnsNames []string, pubkey crypto.PublicKey) string {
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := LoadCertFromFile(FactoryCaCertFile)
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      subject,
		Issuer:       factoryCa.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		IsCA:        false,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    dnsNames,
	}
	return genCertificate(&crtTemplate, factoryCa, pubkey, factoryKey)
}

func genDeviceCaCert(subject pkix.Name, pubkey crypto.PublicKey) string {
	factoryKey := factoryCaKeyStorage.loadKey()
	factoryCa := LoadCertFromFile(FactoryCaCertFile)
	crtTemplate := x509.Certificate{
		SerialNumber: genRandomSerialNumber(),
		Subject:      subject,
		Issuer:       factoryCa.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	return genCertificate(&crtTemplate, factoryCa, pubkey, factoryKey)
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

func genRandomSerialNumber() *big.Int {
	// Generate a 160 bits serial number (20 octets)
	max := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(160), nil)
	serial, err := rand.Int(rand.Reader, max)
	subcommands.DieNotNil(err)
	return serial
}
