package keys

import (
	"crypto/ecdsa"
	cryptoX509 "crypto/x509"
	"encoding/pem"
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
		Use:   "re-sign-device-ca <PKI Directory> [<Old PKI Directory>]",
		Short: "Re-sign all existing Device CAs with a new root CA for your Factory PKI",
		Run:   doReSignDeviceCaRenewal,
		Args:  cobra.RangeArgs(1, 2),
		Long: `Re-sign all existing Device CAs with a new root CA for your Factory PKI.

Both currently active and disabled Device CAs are being re-signed.
All their properties are preserved, including a serial number.
Only the signature and authority key ID are being changed.
This allows old certificates (issued by a previous root CA) to continue being used to issue device client certificates.

Re-signed device CA certificates are stored in the provided PKI directory.
An old PKI directory is used to locate corresponding private keys, and copy them into the PKI directory.
Each located device CA gets the same file name, as it was in the old PKI directory.
If a device CA certificate cannot be located in an old PKI directory - it does not get stored locally.
If an old PKI directory argument is not provided, new certificates are not stored locally.
`,
	}
	caRenewalCmd.AddCommand(cmd)
	addStandardHsmFlags(cmd)
}

func doReSignDeviceCaRenewal(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	certsDir := args[0]

	hsm, err := validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel)
	subcommands.DieNotNil(err)
	x509.InitHsm(hsm)

	var oldKeys []fsKey
	if len(args) > 1 {
		oldCertsDir := args[1]
		cwd, err := os.Getwd()
		subcommands.DieNotNil(err)
		subcommands.DieNotNil(os.Chdir(oldCertsDir))
		oldKeys = loadEcKeys()
		subcommands.DieNotNil(os.Chdir(cwd))
	}
	subcommands.DieNotNil(os.Chdir(certsDir))
	newKeys := loadEcKeys()

	fmt.Println("Fetching a list of existing device CAs")
	resp, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)

	// A safety measure to make sure that the provided PKI directory belongs to the currently active Root CA.
	// This helps preventing an accidental file rewrite in the "wrong" directory.
	fsRootCAData, err := os.ReadFile(x509.FactoryCaCertFile)
	subcommands.DieNotNil(err, "Could not load a Root CA cert file")
	if certs := parseCertList(string(fsRootCAData)); len(certs) != 1 {
		subcommands.DieNotNil(fmt.Errorf("There must be exactly one cert in a Root CA file: %s", x509.FactoryCaCertFile))
	} else if resp.ActiveRoot != certs[0].SerialNumber.Text(10) {
		subcommands.DieNotNil(fmt.Errorf("A provided PKI directory is not a currently active factory PKI: %s", certsDir))
	}

	fmt.Println("Re-signing existing device CAs using a new Root CA key")
	certs := client.CaCerts{}
	storedFiles := make(map[string][2]string, 0)
	missingFiles := make([]string, 0)
	for _, ca := range parseCertList(resp.CaCrt) {
		newCaPem := x509.ReSignCrt(ca)
		if len(certs.CaCrt) > 0 {
			certs.CaCrt += "\n"
		}
		certs.CaCrt += newCaPem

		// Locate cert/key files by comparing public keys of a server-side cert and a private key file.
		// This is the most reliable (and secure) way to identify corresponding key files.
		var found bool
		serial := ca.SerialNumber.Text(10)
		for _, key := range newKeys {
			if key.pubkey.Equal(ca.PublicKey) {
				// A key file is already present in a new PKI folder. Simply overwrite a cert file.
				certFile := strings.TrimSuffix(key.filename, ".key") + ".pem"
				subcommands.DieNotNil(os.WriteFile(certFile, []byte(newCaPem), 0400))
				storedFiles[serial] = [2]string{certFile, key.filename}
				found = true
				break
			}
		}

		if found {
			continue
		}
		found = false
		for _, key := range oldKeys {
			if key.pubkey.Equal(ca.PublicKey) {
				// A key file is found in an old PKI folder. Copy a private key and write a cert file.
				certFile := strings.TrimSuffix(key.filename, ".key") + ".pem"
				subcommands.DieNotNil(os.WriteFile(key.filename, key.content, 0400))
				subcommands.DieNotNil(os.WriteFile(certFile, []byte(newCaPem), 0400))
				storedFiles[serial] = [2]string{certFile, key.filename}
				found = true
				break
			}
		}
		if !found {
			missingFiles = append(missingFiles, serial)
		}
	}

	fmt.Println("Uploading re-signed certs to Foundries.io")
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))

	if len(storedFiles) > 0 {
		fmt.Println("Stored the following Device CA files into a new PKI directory:")
		for serial, filenames := range storedFiles {
			fmt.Printf("\t- Serial %s -> %s, %s\n", serial, filenames[0], filenames[1])
		}
	}
	if len(missingFiles) > 0 {
		fmt.Printf(`Could not find private key files for the following Device CA serials:
	- %s
Corresponding Device CA certificates were not stored on your filesystem.
You may copy those private key files manually from an old PKI directory.
Alternatively, you can re-run this command again providing an old PKI directory containing these files.
`, strings.Join(missingFiles, ", "))
	}
}

type fsKey struct {
	filename string
	content  []byte
	pubkey   ecdsa.PublicKey
}

func loadEcKeys() []fsKey {
	dir, err := os.Getwd()
	subcommands.DieNotNil(err)
	files, err := os.ReadDir(dir)
	subcommands.DieNotNil(err)
	res := make([]fsKey, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() || !strings.HasSuffix(file.Name(), ".key") {
			continue
		}
		data, err := os.ReadFile(file.Name())
		subcommands.DieNotNil(err, "Failed to open key file: "+file.Name())
		block, rest := pem.Decode(data)
		if block == nil || len(rest) > 0 {
			// This might be some custom user's key file format; skip it.
			continue
		}
		key, err := cryptoX509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			// This might be a non-EC key file; skip it.
			continue
		}
		res = append(res, fsKey{file.Name(), data, key.PublicKey})
	}
	return res
}
