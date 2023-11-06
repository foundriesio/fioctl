package x509

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/foundriesio/fioctl/subcommands"
)

const (
	FactoryCaKeyFile  string = "factory_ca.key"
	FactoryCaKeyLabel string = "root-ca"
	FactoryCaCertFile string = "factory_ca.pem"
	DeviceCaKeyFile   string = "local-ca.key"
	DeviceCaCertFile  string = "local-ca.pem"
	TlsCertFile       string = "tls-crt"
	OnlineCaCertFile  string = "online-crt"

	factoryCaName string = "Factory-CA"
)

const (
	// CRL constants for device CA revokation.

	// Revoke the device CA, so that no device client certificates issued by this CA are recognized.
	// Note: the API will revoke the CA for any reason other than 6 or 8.
	// This action is permanent.
	CrlCaRevoke = 9 // RFC 5280 - privilegeWithdrawn

	// Disable the device CA, so that no new devices can be registered with client certificates issued by this CA.
	// Devices that were already registered can still connect and operate as normal.
	// This action can be reverted by CrlCaRenew.
	CrlCaDisable = 6 // RFC 5280 - certificateHold

	// Renew the previously disabled device CA, so that new device registrations are possible again.
	// This value is currently not used by Fioctl. It is here for the reference of API integrators.
	CrlCaRenew = 8 // RFC 5280 - removeFromCRL
)

func readFile(filename string) string {
	data, err := os.ReadFile(filename)
	subcommands.DieNotNil(err)
	return string(data)
}

func writeFile(filename, contents string) {
	err := os.WriteFile(filename, []byte(contents), 0400)
	subcommands.DieNotNil(err)
}

type HsmInfo struct {
	Module     string
	Pin        string
	TokenLabel string
}

type fileStorage struct {
	Filename string
}

type hsmStorage struct {
	HsmInfo
	Label string
}

var factoryCaKeyStorage KeyStorage = &fileStorage{FactoryCaKeyFile}

func InitHsm(hsm *HsmInfo) {
	if hsm != nil {
		factoryCaKeyStorage = &hsmStorage{*hsm, FactoryCaKeyLabel}
	}
}

func ValidateHsmArgs(hsmModule, hsmPin, hsmTokenLabel, moduleArg, pinArg, tokenArg string) (*HsmInfo, error) {
	if len(hsmModule) > 0 {
		if len(hsmPin) == 0 {
			return nil, fmt.Errorf("%s is required with %s", pinArg, moduleArg)
		}
		if len(hsmTokenLabel) == 0 {
			return nil, fmt.Errorf("%s is required with %s", tokenArg, moduleArg)
		}
		return &HsmInfo{Module: hsmModule, Pin: hsmPin, TokenLabel: hsmTokenLabel}, nil
	}
	return nil, nil
}

func parseOnePemBlock(pemBlock string) *pem.Block {
	first, rest := pem.Decode([]byte(pemBlock))
	if first == nil || len(rest) > 0 {
		subcommands.DieNotNil(errors.New("Malformed PEM data"))
	}
	return first
}

func LoadCertFromFile(fn string) *x509.Certificate {
	crtPem := parseOnePemBlock(readFile(fn))
	crt, err := x509.ParseCertificate(crtPem.Bytes)
	subcommands.DieNotNil(err)
	return crt
}
