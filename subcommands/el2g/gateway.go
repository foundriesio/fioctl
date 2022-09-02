package el2g

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	pkiDir string
)

func init() {
	configCmd := &cobra.Command{
		Use:   "config-device-gateway",
		Short: "Setup EdgeLock 2Go support for device gateway",
		Run:   doDeviceGateway,
		Example: `
  fioctl el2g config-device-gateway --pki-dir /tmp/factory-pki`,
	}
	cmd.AddCommand(configCmd)
	configCmd.Flags().StringVarP(&pkiDir, "pki-dir", "", "", "Directory container factory PKI keys")
}

func doDeviceGateway(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	path := filepath.Join(pkiDir, "factory_ca.key")
	_, err := os.Stat(path)
	subcommands.DieNotNil(err)
	path = filepath.Join(pkiDir, "sign_ca_csr")
	_, err = os.Stat(path)
	subcommands.DieNotNil(err)

	ca, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	tmpfile, err := os.CreateTemp("", "el2g-*.csr")
	subcommands.DieNotNil(err)
	defer os.Remove(tmpfile.Name())

	fmt.Println("Requesting CSR from EdgeLock 2Go")
	csr, err := api.El2gCreateDg(factory)
	subcommands.DieNotNil(err)

	fmt.Println("Signing CSR")
	_, err = tmpfile.Write([]byte(csr.Value))
	subcommands.DieNotNil(err)
	sign := exec.Command("./sign_ca_csr", tmpfile.Name())
	sign.Dir = pkiDir
	var out bytes.Buffer
	sign.Stdout = &out
	sign.Stderr = os.Stderr
	subcommands.DieNotNil(sign.Run())

	fmt.Println("Uploading signed certificate")
	subcommands.DieNotNil(api.El2gUploadDgCert(factory, csr.Id, ca.RootCrt, out.String()))

	fmt.Println("Updating factory allowed CA's with")
	fmt.Println(out.String())
	newCa := ca.CaCrt + "\n" + out.String()
	certs := client.CaCerts{CaCrt: newCa}
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
