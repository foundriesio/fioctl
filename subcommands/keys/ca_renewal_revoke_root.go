package keys

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

var serialToRevoke string

func init() {
	cmd := &cobra.Command{
		Use:   "revoke-root <PKI Directory>",
		Short: "Revoke an inactive root CA for your Factory PKI",
		Run:   doRevokeRootRenewal,
		Args:  cobra.ExactArgs(1),
		Long: `Revoke an inactive root CAA for your Factory PKI.

This command removes a Root CA with a given serial and associated cross-signed CA certificates.
After that point a given Root CA can no longer be re-activated.
Devices which were not updated with a new Root CA will instantly lose connectivity to Foundries.io services.

This action is irreversible, so plan properly as to when you perform it.
It is usually the last step of the Root CA rotation, which is optional unless your old Root CA was compromised.`,
	}
	caRenewalCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&serialToRevoke, "serial", "", "",
		"A serial number of the Root CA to revoke in hexadecimal format")
	_ = cmd.MarkFlagRequired("serial")
	addStandardHsmFlags(cmd)
}

func doRevokeRootRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	certsDir := args[0]
	subcommands.DieNotNil(os.Chdir(certsDir))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	toRevoke := map[string]int{serialToRevoke: x509.CrlRootRevoke}
	// The API will determine and revoke cross-signed certificates as well.
	// It will also check that a user is not revoking a currently active Root CA.
	req := client.CaCerts{RootRevokeCrl: x509.CreateCrl(toRevoke)}
	subcommands.DieNotNil(api.FactoryPatchCA(factory, req))
}
