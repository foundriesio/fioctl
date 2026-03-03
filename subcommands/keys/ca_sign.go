package keys

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

var (
	pkiDir string
)

func init() {
	cmd := &cobra.Command{
		Use:   "sign <CSR>",
		Short: "Sign a certificate signing request with the Factory root CA",
		Long: `Sign the given certificate signing request (CSR) using the factory_ca.key
from the specified PKI directory. The signed certificate is printed to stdout.

The certificate type is determined automatically from the CSR's requested
extensions. If the CSR contains a Basic Constraints extension with CA:TRUE,
a CA certificate is produced. Otherwise, a TLS server certificate is produced.`,
		Run:  doSignCa,
		Args: cobra.ExactArgs(1),
	}
	caCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&pkiDir, "pki-dir", "", "", "Path to the PKI directory containing factory_ca.key and factory_ca.pem")
	_ = cmd.MarkFlagRequired("pki-dir")
	_ = cmd.MarkFlagDirname("pki-dir")
}

func doSignCa(cmd *cobra.Command, args []string) {
	csrFile := args[0]

	csrPem, err := os.ReadFile(csrFile)
	subcommands.DieNotNil(err)

	subcommands.DieNotNil(os.Chdir(pkiDir))

	crtPem := x509.SignCsr(string(csrPem))
	fmt.Print(crtPem)
}
