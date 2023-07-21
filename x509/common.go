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
	TlsCsrFile        string = "tls-csr"
	OnlineCaCertFile  string = "online-crt"
	OnlineCaCsrFile   string = "ca-csr"

	CreateCaScript       string = "create_ca"
	CreateDeviceCaScript string = "create_device_ca"
	SignCaScript         string = "sign_ca_csr"
	SignTlsScript        string = "sign_tls_csr"
)

func readFile(filename string) string {
	data, err := os.ReadFile(filename)
	subcommands.DieNotNil(err)
	return string(data)
}
