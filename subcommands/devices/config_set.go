package devices

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	setConfigCmd := &cobra.Command{
		Use:   "set <device> <file1=content> <file2=content ...>",
		Short: "Create a secure configuration for the device",
		Long: `Creates a secure configuration for device encrypting the contents each
file using the device's public key. The fioconfig daemon running
on each device will then be able to grab the latest version of the
device's configuration and apply it.

Basic use can be done with command line arguments. eg:

  fioctl device config set my-device npmtok="root"  githubtok="1234"

The device configuration format also allows specifying what command
to run after a configuration file is updated on the device. To take
advantage of this, the "--raw" flag must be used. eg::

  cat >tmp.json <<EOF
  {
    "reason": "I want to use the on-changed attribute",
    "files": [
      {
        "name": "npmtok",
        "value": "root",
        "on-changed": ["/usr/bin/touch", "/tmp/npmtok-changed"]
      },
      {
        "name": "A-Readable-Value",
        "value": "This won't be encrypted and will be visible from the API",
        "unencrypted": true
      },
      {
        "name": "githubok",
        "value": "1234"
      }
    ]
  }
  > EOF
  fioctl devices config set my-device --raw ./tmp.json

fioctl will read in tmp.json, encrypt its contents, and upload it
to the OTA server. Instead of using ./tmp.json, the command can take
a "-" and will read the content from STDIN instead of a file.
`,
		Run:  doConfigSet,
		Args: cobra.MinimumNArgs(2),
	}
	configCmd.AddCommand(setConfigCmd)
	setConfigCmd.Flags().StringP("reason", "m", "", "Add a message to store as the \"reason\" for this change")
	setConfigCmd.Flags().BoolP("raw", "", false, "Use raw configuration file")
	setConfigCmd.Flags().BoolP("create", "", false, "Replace the whole config with these values. Default is to merge these values in with the existing config values")
}

func loadEciesPub(pubkey string) *ecies.PublicKey {
	block, _ := pem.Decode([]byte(pubkey))
	if block == nil {
		fmt.Println("ERROR: Failed to parse certificate PEM")
		os.Exit(1)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse DER encoded public key: %v\n", err)
	}

	ecpub := pub.(*ecdsa.PublicKey)
	return ecies.ImportECDSAPublic(ecpub)
}

func eciesEncrypt(content string, pubkey *ecies.PublicKey) string {
	message := []byte(content)
	enc, err := ecies.Encrypt(rand.Reader, pubkey, message, nil, nil)
	if err != nil {
		fmt.Printf("ERROR: Cant encrypt: %v\n", err)
	}
	return base64.StdEncoding.EncodeToString(enc)
}

func doConfigSet(cmd *cobra.Command, args []string) {
	name := args[0]
	reason, _ := cmd.Flags().GetString("reason")
	isRaw, _ := cmd.Flags().GetBool("raw")
	shouldCreate, _ := cmd.Flags().GetBool("create")

	logrus.Debugf("Creating new device config for %s", name)
	// Ensure the device has a public key we can encrypt with
	device, err := api.DeviceGet(name)
	subcommands.DieNotNil(err)
	if len(device.PublicKey) == 0 {
		subcommands.DieNotNil(fmt.Errorf("Device has no public key to encrypt with"))
	}
	pubkey := loadEciesPub(device.PublicKey)

	subcommands.SetConfig(&subcommands.SetConfigOptions{
		FileArgs:  args[1:],
		Reason:    reason,
		IsRawFile: isRaw,
		SetFunc: func(cfg client.ConfigCreateRequest) error {
			if shouldCreate {
				return api.DeviceCreateConfig(device.Name, cfg)
			} else {
				return api.DevicePatchConfig(device.Name, cfg, false)
			}
		},
		EncryptFunc: func(value string) string {
			return eciesEncrypt(value, pubkey)
		},
	})
}
