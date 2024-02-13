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
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	factoryCaString := genCertificate(&crtTemplate, &crtTemplate, priv.Public(), priv)
	writeFile(FactoryCaCertFile, factoryCaString)
	return factoryCaString
}

func CreateDeviceCa(cn, ou string) string {
	return CreateDeviceCaExt(cn, ou, DeviceCaKeyFile, DeviceCaCertFile)
}

func CreateDeviceCaExt(cn, ou, keyFile, certFile string) string {
	priv := genAndSaveKeyToFile(keyFile)
	crtPem := genCaCert(marshalSubject(cn, ou), priv.Public())
	writeFile(certFile, crtPem)
	return crtPem
}

func SignCaCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	crtPem := genCaCert(csr.Subject, csr.PublicKey)
	writeFile(OnlineCaCertFile, crtPem)
	return crtPem
}

func SignEl2GoCsr(csrPem string) string {
	csr := parsePemCertificateRequest(csrPem)
	return genCaCert(csr.Subject, csr.PublicKey)
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

func genCaCert(subject pkix.Name, pubkey crypto.PublicKey) string {
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
