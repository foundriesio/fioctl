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

func CreateFactoryCa(ou string) string {
	run("./" + CreateCaScript)
	return readFile(FactoryCaCertFile)
}

func CreateDeviceCa(cn string, ou string) string {
	run("./"+CreateDeviceCaScript, DeviceCaKeyFile, DeviceCaCertFile)
	return readFile(DeviceCaCertFile)
}

func SignTlsCsr(csrPem string) string {
	return run("./"+SignTlsScript, TlsCsrFile)
}

func SignCaCsr(csrPem string) string {
	return run("./"+SignCaScript, OnlineCaCsrFile)
}

func SignEl2GoCsr(csrPem string) string {
	tmpFile, err := os.CreateTemp("", "el2g-*.csr")
	subcommands.DieNotNil(err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	_, err = tmpFile.Write([]byte(csrPem))
	subcommands.DieNotNil(err)
	return run("./"+SignCaScript, tmpFile.Name())
}
