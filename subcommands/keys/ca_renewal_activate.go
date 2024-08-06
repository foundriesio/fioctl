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
		Use:   "activate <PKI Directory>",
		Short: "Activate a new root CA for your Factory PKI",
		Run:   doActivateCaRenewal,
		Args:  cobra.ExactArgs(1),
		Long: `Activate a new root CA for your Factory PKI.

This commands activates a new root CA by superseding a previous one.
An active root CA is the one used to issue, sign, and verify all device CAs and TLS certificates.
There can be many root CAs, recognized as valid by devices.
But, only one of them can be active at a time.

This command can be used many times to switch between currently active root CAs.
Cross-signed root CA certificates (which are a part of a renewal bundle) cannot be activated.

Typically, this command is used after deploying the renewal bundle to all devices, and before re-signing device CAs.`,
	}
	caRenewalCmd.AddCommand(cmd)
	addStandardHsmFlags(cmd)
}

func doActivateCaRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	certsDir := args[0]
	subcommands.DieNotNil(os.Chdir(certsDir))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	newRootOnDisk := x509.LoadCertFromFile(x509.FactoryCaCertFile)
	newCerts := client.CaCerts{ActiveRoot: newRootOnDisk.SerialNumber.Text(10)}
	oldCerts, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)
	if oldCerts.ActiveRoot == newCerts.ActiveRoot {
		subcommands.DieNotNil(fmt.Errorf("A given root CA with serial %s is already active", oldCerts.ActiveRoot))
	}

	toRevoke := map[string]int{oldCerts.ActiveRoot: x509.CrlRootSupersede}
	newCerts.RootRevokeCrl = x509.CreateCrl(toRevoke)
	subcommands.DieNotNil(api.FactoryPatchCA(factory, newCerts))
}
