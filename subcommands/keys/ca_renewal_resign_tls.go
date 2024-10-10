package keys

import (
	"errors"
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
		Use:   "re-sign-tls <PKI Directory>",
		Short: "Re-sign the TLS certificates used by Device Gateway, OSTree Server, and EST Server, if applicable",
		Run:   doReSignTlsRenewal,
		Args:  cobra.ExactArgs(1),
	}
	caRenewalCmd.AddCommand(cmd)
	addStandardHsmFlags(cmd)
}

func doReSignTlsRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Requesting existing TLS certificates")
	resp, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	fmt.Println("Re-signing Foundries TLS certificate")
	certs := parseCertList(resp.TlsCrt)
	if len(certs) != 1 {
		subcommands.DieNotNil(errors.New("There must be exactly one TLS certificate"))
	}
	req := client.CaCerts{TlsCrt: x509.ReSignCrt(certs[0])}
	subcommands.DieNotNil(os.WriteFile(x509.TlsCertFile, []byte(req.TlsCrt), 0400))

	if len(resp.EstCrt) > 0 {
		fmt.Println("Re-signing Foundries EST certificate")
		certs := parseCertList(resp.EstCrt)
		if len(certs) != 1 {
			subcommands.DieNotNil(errors.New("There must be zero or one EST certificate"))
		}
		req.EstCrt = x509.ReSignCrt(certs[0])
	}

	fmt.Println("Uploading signed certs to Foundries")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, req))
}
