package el2g

import (
	"fmt"
	"os"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	pkiDir        string
	hsmModule     string
	hsmPin        string
	hsmTokenLabel string
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
	configCmd.Flags().StringVarP(&pkiDir, "pki-dir", "", "", "Directory containing factory PKI keys")
	configCmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Load a root CA key from a PKCS#11 compatible HSM using this module")
	configCmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	configCmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token containing the root CA key")
	_ = configCmd.MarkFlagRequired("pki-dir")
}

func doDeviceGateway(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	subcommands.DieNotNil(os.Chdir(pkiDir))
	hsm, err := x509.ValidateHsmArgs(
		hsmModule, hsmPin, hsmTokenLabel, "--hsm-module", "--hsm-pin", "--hsm-token-label")
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	ca, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	fmt.Println("Requesting CSR from EdgeLock 2Go")
	csr, err := api.El2gCreateDg(factory)
	subcommands.DieNotNil(err)

	fmt.Println("Signing CSR")
	generatedCa := x509.SignEl2GoCsr(csr.Value)

	fmt.Println("Uploading signed certificate")
	errPrefix := "Unable to upload certificate:\n" + generatedCa
	subcommands.DieNotNil(api.El2gUploadDgCert(factory, csr.Id, ca.RootCrt, generatedCa), errPrefix)

	fmt.Println("Updating factory allowed CA's with")
	fmt.Println(generatedCa)
	newCa := ca.CaCrt + "\n" + generatedCa
	certs := client.CaCerts{CaCrt: newCa}
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
