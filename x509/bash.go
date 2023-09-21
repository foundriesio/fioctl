//go:build !windows

package x509

import (
	"os"
	"os/exec"

	"github.com/foundriesio/fioctl/subcommands"
)

func run(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	subcommands.DieNotNil(err)
	return string(out)
}

func CreateFactoryCa(storage KeyStorage, ou string) string {
	storage.setEnvVariables()
	run("./" + CreateCaScript)
	return string(readFile(FactoryCaCertFile))
}

func CreateDeviceCa(_ KeyStorage, cn string, ou string) string {
	run("./"+CreateDeviceCaScript, DeviceCaKeyFile, DeviceCaCertFile)
	return string(readFile(DeviceCaCertFile))
}

func SignTlsCsr(_ KeyStorage, csrPem string) string {
	return run("./"+SignTlsScript, TlsCsrFile)
}

func SignCaCsr(_ KeyStorage, csrPem string) string {
	return run("./"+SignCaScript, OnlineCaCsrFile)
}

func SignEl2GoCsr(_ KeyStorage, csrPem string) string {
	tmpFile, err := os.CreateTemp("", "el2g-*.csr")
	subcommands.DieNotNil(err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	_, err = tmpFile.Write([]byte(csrPem))
	subcommands.DieNotNil(err)
	return run("./"+SignCaScript, tmpFile.Name())
}

type KeyStorage interface {
	setEnvVariables()
}

type KeyStorageHsm struct {
	Hsm HsmInfo
}

func (s *KeyStorageHsm) setEnvVariables() {
	os.Setenv("HSM_MODULE", s.Hsm.Module)
	os.Setenv("HSM_PIN", s.Hsm.Pin)
	os.Setenv("HSM_TOKEN_LABEL", s.Hsm.TokenLabel)
}

type KeyStorageFiles struct{}

func (s *KeyStorageFiles) setEnvVariables() {}
