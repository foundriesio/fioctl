package devices

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"

	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var (
	configReason string
	configRaw    bool
	configCreate bool
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
advantage of this, the "--raw" flag must be used. eg:

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
to the OTA server.
`,
		Run:  doConfigSet,
		Args: cobra.MinimumNArgs(2),
	}
	configCmd.AddCommand(setConfigCmd)
	setConfigCmd.Flags().StringVarP(&configReason, "reason", "m", "", "Add a message to store as the \"reason\" for this change")
	setConfigCmd.Flags().BoolVarP(&configRaw, "raw", "", false, "Use raw configuration file")
	setConfigCmd.Flags().BoolVarP(&configCreate, "create", "", false, "Replace the whole config with these values. Default is to merge these values in with the existing config values")
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

func loadConfig(configFile string, cfg *client.ConfigCreateRequest) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("ERROR: Unable to read config file: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(content, cfg); err != nil {
		fmt.Printf("ERROR: Unable to parse config file: %v\n", err)
		os.Exit(1)
	}
}

func doConfigSet(cmd *cobra.Command, args []string) {
	logrus.Debug("Creating new device config")

	// Ensure the device has a public key we can encrypt with
	device, err := api.DeviceGet(args[0])
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	if len(device.PublicKey) == 0 {
		fmt.Println("ERROR: Device has no public key to encrypt with")
		os.Exit(1)
	}

	pubkey := loadEciesPub(device.PublicKey)

	cfg := client.ConfigCreateRequest{Reason: configReason}
	if configRaw {
		loadConfig(args[1], &cfg)
		for i := range cfg.Files {
			if !cfg.Files[i].Unencrypted {
				cfg.Files[i].Value = eciesEncrypt(cfg.Files[i].Value, pubkey)
			}
		}
	} else {
		for _, keyval := range args[1:] {
			parts := strings.SplitN(keyval, "=", 2)
			if len(parts) != 2 {
				fmt.Println("ERROR: Invalid file=content argument: ", keyval)
				os.Exit(1)
			}
			enc := eciesEncrypt(parts[1], pubkey)
			cfg.Files = append(cfg.Files, client.ConfigFile{Name: parts[0], Value: enc})
		}
	}

	if configCreate {
		if err := api.DeviceCreateConfig(device.Name, cfg); err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		if err := api.DevicePatchConfig(device.Name, cfg); err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
