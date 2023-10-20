package keys

import (
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

func init() {
	cmd := &cobra.Command{
		Use:   "revoke-device-ca <PKI Directory>",
		Short: "Revoke device CA, so that devices with client certificates it issued can no longer connect to your Factory",
		Run:   doRevokeDeviceCa,
		Args:  cobra.ExactArgs(1),
		Long: `Revoke device CA, so that devices with client certificates it issued can no longer connect to your Factory.

When the online or local device CA is revoked:
- It is no longer possible to register new devices with client certificates it had issued.
- Existing devices with client certificates it issued can no longer connect to your Factory.

You may later re-add a revoked device CA using "keys ca update", if you still keep its certificate stored somewhere.
Once you do this, devices with client certificates issued by this device CA may connect to your Factory again.`,
		Example: `
# Revoke a local device CA by providing a (default) file name inside your PKI directory:
fioctl keys ca revoke-device-ca /path/to/pki/dir --ca-file local-ca

# Revoke two local device CAs given a full path to their files:
fioctl keys ca revoke-device-ca /path/to/pki/dir --ca-file /path/to/ca1.pem --ca-file /path/to/ca2.crt

# Revoke two device CAs given their serial numbers:
fioctl keys ca revoke-device-ca /path/to/pki/dir --ca-serial <base-10-serial-1> --ca-file <base-10-serial-2>

# Revoke a local device CA, when your factory root CA private key is stored on an HSM:
fioctl keys ca revoke-device-ca /path/to/pki/dir --ca-file local-ca \
  --hsm-module /path/to/pkcs11-module.so --hsm-pin 1234 --hsm-token-label <token-label-for-key>

# Show a generated CRL that would be sent to the server to revoke a local device CA, without actually revoking it.
fioctl keys ca revoke-device-ca /path/to/pki/dir --ca-file local-ca --dry-run --pretty`,
	}
	caCmd.AddCommand(cmd)
	cmd.Flags().BoolP("dry-run", "", false,
		"Do not revoke the certificate, but instead show a generated CRL that will be uploaded to the server.")
	cmd.Flags().BoolP("pretty", "", false,
		"Can be used with dry-run to show the generated CRL in a pretty format.")
	cmd.Flags().StringArrayP("ca-file", "", nil,
		"A file name of the device CA to revoke. Can be used multiple times to revoke several device CAs.")
	cmd.Flags().StringArrayP("ca-serial", "", nil,
		"A serial number (base 10) of the device CA to revoke. Can be used multiple times to revoke several device CAs.")
	_ = cmd.MarkFlagFilename("ca-file")
	// HSM variables defined in ca_create.go
	cmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Load a root CA key from a PKCS#11 compatible HSM using this module")
	cmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	cmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token containing the root CA key")
}

func doRevokeDeviceCa(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	// TODO: Implement the --dry-run and --pretty arguments
	// dryRun, _ := cmd.Flags().GetBool("dry-run")
	// pretty, _ := cmd.Flags().GetBool("pretty")
	caFiles, _ := cmd.Flags().GetStringArray("ca-file")
	caSerials, _ := cmd.Flags().GetStringArray("ca-serial")

	if len(caFiles)+len(caSerials) == 0 {
		subcommands.DieNotNil(errors.New("At least one of --ca-file or --ca-serial must be provided"))
	}

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := x509.ValidateHsmArgs(
		hsmModule, hsmPin, hsmTokenLabel, "--hsm-module", "--hsm-pin", "--hsm-token-label")
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Generating Certificate Revocation List")
	toRevoke := make(map[string]int, len(caSerials)+len(caFiles))
	for _, serial := range caSerials {
		num := new(big.Int)
		if _, ok := num.SetString(serial, 10); !ok {
			subcommands.DieNotNil(fmt.Errorf("Value %s is not a valid base 10 serial", serial))
		}
		toRevoke[serial] = x509.CrlCaRevoke
	}
	for _, filename := range caFiles {
		ca := x509.LoadCertFromFile(filename)
		toRevoke[ca.SerialNumber.Text(10)] = x509.CrlCaRevoke
	}
	fmt.Println("Signing CRL by factory root CA")
	certs := client.CaCerts{CaRevokeCrl: x509.CreateCrl(toRevoke)}
	fmt.Println("Uploading CRL to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
