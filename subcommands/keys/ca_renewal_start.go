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

var (
	hsmOldModule     string
	hsmOldPin        string
	hsmOldTokenLabel string
)

func init() {
	cmd := &cobra.Command{
		Use:   "start <New PKI Directory> <Old PKI Directory>",
		Short: "Start a root of trust renewal for your factory PKI",
		Run:   doStartCaRenewal,
		Args:  cobra.ExactArgs(2),
		Long: `Start a root of trust renewal for your factory PKI.

This command does a few things:.
1. First, it generates a new root of trust for your factory.
2. Second, it cross-signs a new root of trust using an old root of trust to prepare a CA renewal bundle.
   This CA renewal bundle is compliant with the EST standard, necessary for a secure root CA update on devices.
3. Finally, it uploads the CA renewal bundle to the backend API for validation and storage.

A new root of trust needs to be stored in a separate directory from the previous root of trust.
By the end of the renewal process, all necessary PKI pieces are migrated into this new directory.
If you are using an HSM device - a new private key should be stored under a different label, possibly a new device.
This extreme level of isolation is necessary to prevent an accidental rewrite of the old root of trust.

Once this command completes successfully, a root of trust renewal process is started.`,
	}
	caRenewalCmd.AddCommand(cmd)
	// HSM variable descriptions differ from the standard defined in addStandardHsmFlags
	cmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Create a root CA key on a PKCS#11 compatible HSM using this module")
	cmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	cmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token created for the root CA key")
	cmd.Flags().StringVarP(&hsmOldModule, "hsm-old-module", "", "",
		"A PKCS#11 compatible HSM module storing an old root CA key. By default use --hsm-module.")
	cmd.Flags().StringVarP(&hsmOldPin, "hsm-old-pin", "", "",
		"The PKCS#11 PIN to log into the HSM storing an old root CA key. By default use --hsm-pin.")
	cmd.Flags().StringVarP(&hsmOldTokenLabel, "hsm-old-token-label", "", "",
		"The label of the HSM token containing an old root CA key.")
}

func doStartCaRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	newCertsDir := args[0]
	oldCertsDir := args[1]
	if oldCertsDir == newCertsDir {
		subcommands.DieNotNil(errors.New(`A new PKI directory must be different that an old PKI directory.
A root CA renewal migrates your Factory PKI to a new (initially empty) folder.
This ensures that no sensitive data can be accidentally tampered or erased.`))
	}

	cwd, err := os.Getwd()
	subcommands.DieNotNil(err)

	newHsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	oldHsm, err := validateHsmArgs(
		hsmOldModule, hsmOldPin, hsmOldTokenLabel, "--hsm-old-module", "--hsm-old-pin", "--hsm-old-token-label")
	subcommands.DieNotNil(err)
	if hsmModule == hsmOldModule && hsmTokenLabel == hsmOldTokenLabel {
		subcommands.DieNotNil(errors.New(`When using HSM devices, a new private key must be stored on:
- either a new HSM device with the same or a different label;
- or the same HSM device with a different label.
This warrants that an existing private key is not accidentally overwritten or erased.`))
	}

	// Phase 1 - Load existing Root CA and check if it is the active one.
	// Do not check if the user actually has access to its private key - that is verified later by signing certificates.
	fmt.Println("Verifying access to old root CA for Factory")
	subcommands.DieNotNil(os.Chdir(oldCertsDir))
	oldCerts, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err, "Failed to fetch current root CA for Factory")
	oldRootOnDisk := x509.LoadCertFromFile(x509.FactoryCaCertFile)
	oldRootOnDiskSerial := oldRootOnDisk.SerialNumber.Text(10)
	if oldRootOnDiskSerial != oldCerts.ActiveRoot {
		subcommands.DieNotNil(fmt.Errorf(
			"Old PKI directory has root CA with serial %s but %s is expected", oldRootOnDiskSerial, oldCerts.ActiveRoot))
	}

	// Phase 2 - Generate a new Root CA.
	certs := client.CaCerts{RootCrt: oldCerts.RootCrt}
	subcommands.DieNotNil(os.Chdir(cwd))
	subcommands.DieNotNil(os.Chdir(newCertsDir))
	if _, err := os.Stat(x509.FactoryCaCertFile); err == nil {
		subcommands.DieNotNil(fmt.Errorf(
			"Factory CA file already exists inside %s. Cancelling to prevent accidental rewrite", newCertsDir))
	} else if !os.IsNotExist(err) {
		subcommands.DieNotNil(err)
	}
	fmt.Println("Creating new offline root CA for Factory")
	x509.InitHsm(newHsm)
	certs.RootCrt += x509.CreateFactoryCa(factory)
	newRootOnDisk := x509.LoadCertFromFile(x509.FactoryCaCertFile)

	// Phase 2 - Generate 2 cross-signed Root CAs.
	fmt.Println("Generating two cross-signed root CAs for Factory")
	// Old CA cross-signed by a new CA.
	certs.RootCrt += x509.CreateFactoryCrossCa(factory, oldRootOnDisk.PublicKey)

	// New CA cross-signed by an old CA.
	subcommands.DieNotNil(os.Chdir(cwd))
	subcommands.DieNotNil(os.Chdir(oldCertsDir))
	x509.InitHsm(oldHsm)
	certs.RootCrt += x509.CreateFactoryCrossCa(factory, newRootOnDisk.PublicKey)

	fmt.Println("Uploading signed certs to Foundries")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
