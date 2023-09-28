package x509

import (
	"os"

	"github.com/foundriesio/fioctl/subcommands"
)

const (
	FactoryCaKeyFile  string = "factory_ca.key"
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
