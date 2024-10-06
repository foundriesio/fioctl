package keys

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

func init() {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the EST TLS certificate authorized for this factory",
		Run:   doShowEst,
	}
	estCmd.AddCommand(cmd)
	cmd.Flags().BoolVarP(&prettyFormat, "pretty", "", false, "Display human readable output of each certificate")

	cmd = &cobra.Command{
		Use:   "authorize <PKI directory>",
		Short: "Authorize Foundries.io to run an EST server at <repoid>.est.foundries.io",
		Run:   doAuthorizeEst,
		Args:  cobra.ExactArgs(1),
		Long: `This command will initiate a transaction with api.foundries.io that:

  * api.foundries.io will create a new private key and TLS certificate signing request
  * This command will sign the request using the Factory's root key.
  * Upload the resultant TLS certificate to api.foundries.io`,
	}
	estCmd.AddCommand(cmd)
	addStandardHsmFlags(cmd)
}

func doShowEst(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing EST cert for %s", factory)

	cert, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)
	if len(cert.EstCrt) == 0 {
		fmt.Println("EST TLS certificate has not been configured for this factory.")
	} else if prettyFormat {
		prettyPrint(cert.EstCrt)
	} else {
		fmt.Println(cert.EstCrt)
	}
}

func doAuthorizeEst(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	logrus.Debugf("Authorizing EST for %s", factory)
	csrs, err := api.FactoryCreateCA(factory, client.CaCreateOptions{CreateEstCert: true})
	subcommands.DieNotNil(err)

	certs := client.CaCerts{EstCrt: x509.SignEstCsr(csrs.EstCsr)}
	fmt.Println("Uploading new EST certificate:")
	fmt.Println(certs.EstCrt)
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
