package keys

import (
	x509Lib "crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
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

const (
	crlCmdAnnotation      = "crl-revoke"
	crlCmdRevokeDeviceCa  = "revoke-device-ca"
	crlCmdDisableDeviceCa = "disable-device-ca"
)

func init() {
	revokeCmd := &cobra.Command{
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
		Annotations: map[string]string{crlCmdAnnotation: crlCmdRevokeDeviceCa},
	}
	caCmd.AddCommand(revokeCmd)
	addRevokeCmdFlags(revokeCmd, "revoke")

	disableCmd := &cobra.Command{
		Use:   "disable-device-ca <PKI Directory>",
		Short: "Disable device CA, so that new devices with client certificates it issued can no longer be registered",
		Run:   doRevokeDeviceCa,
		Args:  cobra.ExactArgs(1),
		Long: `Disable device CA, so that new devices with client certificates it issued can no longer be registered.

When the online or local device CA is disabled:
- It is no longer possible to register new devices with client certificates it had issued.
- Existing devices with client certificates it issued may continue to connect and use your Factory.

Usually, when the device CA is compromised, a user should:
1. Immediately disable a given device CA using "fioctl keys ca disable-device-ca <PKI Directory> --serial <CA Serial>".
2. Inspect their devices with client certificates issued by that device CA, and remove compromised devices (see "fioctl devices list|delete").
3. Create a new device CA using "fioctl keys ca add-device-ca <PKI Directory> --online-ca|--local-ca".
4. Rotate a client certificate of legitimate devices to the certificate issued by a new device CA (see "fioctl devices config rotate-certs").
5. Revoke a given device CA using "fioctl keys ca revoke-device-ca <PKI Directory> --serial <CA Serial>".`,
		Example: `
# Disable two device CAs given their serial numbers:
fioctl keys ca disable-device-ca /path/to/pki/dir --ca-serial <base-10-serial-1> --ca-file <base-10-serial-2>

# Show a generated CRL that would be sent to the server to disable a local device CA, without actually disabling it.
fioctl keys ca disable-device-ca /path/to/pki/dir --ca-file local-ca --dry-run --pretty

# See "fioctl keys ca revoke-device-ca --help" for more examples; these two commands have a very similar syntax.`,
		Annotations: map[string]string{crlCmdAnnotation: crlCmdDisableDeviceCa},
	}
	caCmd.AddCommand(disableCmd)
	addRevokeCmdFlags(disableCmd, "disable")
}

func addRevokeCmdFlags(cmd *cobra.Command, op string) {
	cmd.Flags().BoolP("dry-run", "", false,
		"Do not "+op+" the certificate, but instead show a generated CRL that will be uploaded to the server.")
	cmd.Flags().BoolP("pretty", "", false,
		"Can be used with dry-run to show the generated CRL in a pretty format.")
	cmd.Flags().StringArrayP("ca-file", "", nil,
		"A file name of the device CA to "+op+". Can be used multiple times to "+op+" several device CAs")
	cmd.Flags().StringArrayP("ca-serial", "", nil,
		"A serial number (base 10) of the device CA to "+op+". Can be used multiple times to "+op+" several device CAs")
	_ = cmd.MarkFlagFilename("ca-file")
	addStandardHsmFlags(cmd)
}

func doRevokeDeviceCa(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	pretty, _ := cmd.Flags().GetBool("pretty")
	caFiles, _ := cmd.Flags().GetStringArray("ca-file")
	caSerials, _ := cmd.Flags().GetStringArray("ca-serial")
	crlReason := map[string]int{
		crlCmdRevokeDeviceCa:  x509.CrlCaRevoke,
		crlCmdDisableDeviceCa: x509.CrlCaDisable,
	}[cmd.Annotations[crlCmdAnnotation]]

	if len(caFiles)+len(caSerials) == 0 {
		subcommands.DieNotNil(errors.New("At least one of --ca-file or --ca-serial must be provided"))
	}
	for _, v := range caFiles {
		assertFileName("--ca-file", v)
	}

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Generating Certificate Revocation List")
	toRevoke := make(map[string]int, len(caSerials)+len(caFiles))
	for _, serial := range caSerials {
		num := new(big.Int)
		if _, ok := num.SetString(serial, 10); !ok {
			subcommands.DieNotNil(fmt.Errorf("Value %s is not a valid base 10 serial", serial))
		}
		toRevoke[serial] = crlReason
	}
	for _, filename := range caFiles {
		ca := x509.LoadCertFromFile(filename)
		toRevoke[ca.SerialNumber.Text(10)] = crlReason
	}

	caList, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)
	validSerials := make(map[string]bool, 0)
	for _, c := range parseCertList(caList.CaCrt) {
		validSerials[c.SerialNumber.Text(10)] = true
	}
	for serial := range toRevoke {
		if _, ok := validSerials[serial]; !ok {
			subcommands.DieNotNil(fmt.Errorf("There is no actual device CA with serial %s", serial))
		}
	}

	fmt.Println("Signing CRL by factory root CA")
	certs := client.CaCerts{CaRevokeCrl: x509.CreateCrl(toRevoke)}

	if dryRun {
		fmt.Println(certs.CaRevokeCrl)
		if pretty {
			prettyPrintCrl(certs.CaRevokeCrl)
		}
		return
	}

	fmt.Println("Uploading CRL to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}

func prettyPrintCrl(crlPem string) {
	block, remaining := pem.Decode([]byte(crlPem))
	if block == nil || len(remaining) > 0 {
		subcommands.DieNotNil(errors.New("Invalid PEM block"), "Failed to parse generated CRL:")
		return // linter
	}
	c, err := x509Lib.ParseRevocationList(block.Bytes)
	subcommands.DieNotNil(err, "Failed to parse generated CRL:")
	fmt.Println("Certificate Revocation List:")
	fmt.Println("\tIssuer:", c.Issuer)
	fmt.Println("\tValidity:")
	fmt.Println("\t\tNot Before:", c.ThisUpdate)
	fmt.Println("\t\tNot After:", c.NextUpdate)
	fmt.Println("\tSignature Algorithm:", c.SignatureAlgorithm)
	fmt.Println("\tSignature:", hex.EncodeToString(c.Signature))
	fmt.Println("\tRevoked Certificates:")
	for _, crt := range c.RevokedCertificates {
		fmt.Println("\t\tSerial:", crt.SerialNumber.Text(10))
		fmt.Println("\t\t\tRevocation Date:", crt.RevocationTime)
		if len(crt.Extensions) > 0 {
			fmt.Println("\t\t\tExtensions:")
			for _, ext := range crt.Extensions {
				if ext.Id.String() == "2.5.29.21" {
					fmt.Print("\t\t\t\tx509v3 Reason Code:")
					if ext.Critical {
						fmt.Print("(critical)")
					}
					var val asn1.Enumerated
					if _, err := asn1.Unmarshal(ext.Value, &val); err != nil {
						fmt.Println(err)
					} else {
						readable := map[int]string{
							x509.CrlCaRevoke:  "Revoke",
							x509.CrlCaDisable: "Disable",
							x509.CrlCaRenew:   "Renew",
						}[int(val)]
						fmt.Println("\n\t\t\t\t\t", readable, "-", val)
					}
				} else {
					fmt.Println("\t\t\t\tUnknown OID", ext.Id.String())
				}
			}
		}
	}
	if len(c.Extensions) > 0 {
		fmt.Println("\tExtensions:")
		for _, ext := range c.Extensions {
			if ext.Id.String() == "2.5.29.35" {
				fmt.Print("\t\tx509v3 Authority Key Id: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println("\n\t\t\t", hex.EncodeToString(c.AuthorityKeyId))
			} else if ext.Id.String() == "2.5.29.20" {
				fmt.Print("\t\tx509v3 CRL Number: ")
				if ext.Critical {
					fmt.Print("(critical)")
				}
				fmt.Println("\n\t\t\t", c.Number)
			} else {
				fmt.Println("\t\tUnknown OID", ext.Id.String())
			}
		}
	}
}
