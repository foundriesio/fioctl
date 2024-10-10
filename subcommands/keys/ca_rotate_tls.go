package keys

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rotate-tls <PKI Directory>",
		Short: "Rotate the TLS certificate used by Device Gateway and OSTree Server",
		Run:   doRotateTls,
		Args:  cobra.ExactArgs(1),
	}
	caCmd.AddCommand(cmd)
	// HSM variables defined in ca_create.go
	cmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Load a root CA key from a PKCS#11 compatible HSM using this module")
	cmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	cmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token containing the root CA key")
}

func doRotateTls(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := x509.ValidateHsmArgs(
		hsmModule, hsmPin, hsmTokenLabel, "--hsm-module", "--hsm-pin", "--hsm-token-label")
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Requesting new Foundries.io TLS CSR")
	csrs, err := api.FactoryCreateCA(factory, client.CaCreateOptions{CreateTlsCert: true})
	subcommands.DieNotNil(err)

	if _, err := os.Stat(x509.TlsCertFile); !os.IsNotExist(err) {
		fmt.Printf("Moving existing TLS cert file from %s to %s.bak", x509.TlsCertFile, x509.TlsCertFile)
		subcommands.DieNotNil(os.Rename(x509.TlsCertFile, x509.TlsCertFile+".bak"))
	}
	fmt.Println("Signing Foundries.io TLS CSR")
	certs := client.CaCerts{TlsCrt: x509.SignTlsCsr(csrs.TlsCsr)}

	fmt.Println("Uploading signed certs to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
