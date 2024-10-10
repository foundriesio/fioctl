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
	addStandardHsmFlags(cmd)
}

func doRotateTls(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
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
