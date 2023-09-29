package x509

import (
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

func InitHsm(hsm HsmInfo) {
	factoryCaKeyStorage = &hsmStorage{hsm, FactoryCaKeyLabel}
}
