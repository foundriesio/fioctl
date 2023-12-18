package keys

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/x509"
)

func init() {
	cmd := &cobra.Command{
		Use:   "add-device-ca <PKI Directory>",
		Short: "Add device CA to the list of CAs allowed to issue device client certificates",
		Run:   doAddDeviceCa,
		Args:  cobra.ExactArgs(1),
		Long: `Add device CA to the list of CAs allowed to issue device client certificates.

This command can add one or both of the following certificates:

### online-ca - A Foundries.io owned keypair to support lmp-device-register.
In order for lmp-device-register to work, Foundries.io needs the ability to sign client certificates for devices.
If enabled, the factory_ca keypair will sign the certificate signing request returned from the API.
If the online-ca was already created earlier, a new online-ca will replace it for the registration process.
Still, the previous online-ca will be present in a list of device CAs trusted by the device gateway,
so that devices with client certificates issued by it may continue to connect to Foundries.io services.

### local-ca - A keypair you own that can be used for things like your manufacturing process,
where you may generate device client certificates without having to communicate with Foundries.io web services.
You can create as many local-ca files as you need, and use each of them to generate device client certificates.
All such CAs will be added to the list of device CAs trusted by the device gateway.`,
	}
	caCmd.AddCommand(cmd)
	cmd.Flags().BoolP("online-ca", "", false,
		"Create an online CA owned by Foundries.io that works with lmp-device-register")
	cmd.Flags().BoolP("local-ca", "", false,
		"Create a local CA that you can use for signing your own device certificates")
	cmd.Flags().StringP("local-ca-filename", "", x509.DeviceCaCertFile,
		fmt.Sprintf("A file name of the local CA (only needed if the %s file already exists)", x509.DeviceCaCertFile))
	_ = cmd.MarkFlagFilename("local-ca-filename")
	// HSM variables defined in ca_create.go
	cmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Load a root CA key from a PKCS#11 compatible HSM using this module")
	cmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	cmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token containing the root CA key")
}

func assertFileName(flagName, value string) {
	if len(value) > 0 {
		if strings.ContainsRune(value, os.PathSeparator) {
			subcommands.DieNotNil(fmt.Errorf("The `%s` argument must be filename and not a path: %s", flagName, value))
		}
	}
}

func doAddDeviceCa(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	createLocalCA, _ := cmd.Flags().GetBool("local-ca")
	createOnlineCA, _ := cmd.Flags().GetBool("online-ca")
	localCaFilename, _ := cmd.Flags().GetString("local-ca-filename")

	if !createLocalCA && !createOnlineCA {
		subcommands.DieNotNil(errors.New("At least one of --online-ca or --local-ca must be true"))
	}

	assertFileName("--local-ca-filename", localCaFilename)

	subcommands.DieNotNil(os.Chdir(args[0]))
	hsm, err := x509.ValidateHsmArgs(
		hsmModule, hsmPin, hsmTokenLabel, "--hsm-module", "--hsm-pin", "--hsm-token-label")
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	fmt.Println("Fetching a list of existing device CAs")
	resp, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)
	certs := client.CaCerts{CaCrt: resp.CaCrt}
	if len(certs.CaCrt) == 0 {
		subcommands.DieNotNil(errors.New("Factory PKI not initialized. Set it up using 'fioctl keys ca create'."))
	}

	if createLocalCA {
		localCaKeyFilename := strings.TrimSuffix(localCaFilename, ".pem") + ".key"
		if _, err := os.Stat(localCaFilename); !os.IsNotExist(err) {
			subcommands.DieNotNil(fmt.Errorf(`A local device CA file %s already exists.
Please specify a different name with --local-ca-filename.`, localCaFilename))
		}
		if _, err := os.Stat(localCaKeyFilename); !os.IsNotExist(err) {
			subcommands.DieNotNil(fmt.Errorf(`A local device CA key file %s already exists.
Please specify a different name with --local-ca-filename.`, localCaKeyFilename))
		}

		fmt.Println("Creating local device CA")
		commonName := getDeviceCaCommonName(factory)
		certs.CaCrt += "\n" + x509.CreateDeviceCaExt(commonName, factory, localCaKeyFilename, localCaFilename)
	}

	if createOnlineCA {
		fmt.Println("Requesting new Foundries.io Online Device CA CSR")
		csrs, err := api.FactoryCreateCA(factory, client.CaCreateOptions{CreateOnlineCa: true})
		subcommands.DieNotNil(err)

		if _, err := os.Stat(x509.OnlineCaCertFile); !os.IsNotExist(err) {
			fmt.Printf("Moving existing online device CA file from %s to %s.bak", x509.OnlineCaCertFile, x509.OnlineCaCertFile)
			subcommands.DieNotNil(os.Rename(x509.OnlineCaCertFile, x509.OnlineCaCertFile+".bak"))
		}

		fmt.Println("Signing Foundries.io CSR for online use")
		certs.CaCrt += "\n" + x509.SignCaCsr(csrs.CaCsr)
	}

	fmt.Println("Uploading signed certs to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
