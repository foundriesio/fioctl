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
		Use:   "re-sign-device-ca <PKI Directory>",
		Short: "Re-sign all existing Device CAs with a new root CA for your Factory PKI",
		Run:   doReSignDeviceCaRenewal,
		Args:  cobra.ExactArgs(1),
		Long: `Re-sign all existing Device CAs with a new root CA for your Factory PKI.

Both currently active and disabled Device CAs are being re-signed.
All their properties are preserved, including a serial number.
Only the signature and authority key ID are being changed.
This allows old certificates (issued by a previous root CA) to continue being used to issue device client certificates.
`,
	}
	caRenewalCmd.AddCommand(cmd)
	addStandardHsmFlags(cmd)
}

func doReSignDeviceCaRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	certsDir := args[0]
	subcommands.DieNotNil(os.Chdir(certsDir))

	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Fetching a list of existing device CAs")
	resp, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	fmt.Println("Re-signing existing device CAs using a new Root CA key")
	certs := client.CaCerts{}
	for _, ca := range parseCertList(resp.CaCrt) {
		if len(certs.CaCrt) > 0 {
			certs.CaCrt += "\n"
		}
		certs.CaCrt += x509.ReSignCrt(ca)
	}

	fmt.Println("Uploading re-signed certs to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
