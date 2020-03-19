package cmd

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var deviceConfigCmd = &cobra.Command{
	Use:   "config <device> file=content <file2=content ...>",
	Short: "Create a new configuration for the device",
	Run:   doDeviceConfig,
	Args:  cobra.MinimumNArgs(2),
}

var configReason string

func init() {
	deviceCmd.AddCommand(deviceConfigCmd)
	deviceConfigCmd.Flags().StringVarP(&configReason, "reason", "m", "", "Add a message to store as the \"reason\" for this change")
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

func doDeviceConfig(cmd *cobra.Command, args []string) {
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
	for _, keyval := range args[1:] {
		parts := strings.SplitN(keyval, "=", 2)
		if len(parts) != 2 {
			fmt.Println("ERROR: Invalid file=content argument: ", keyval)
			os.Exit(1)
		}
		enc := eciesEncrypt(parts[1], pubkey)
		cfg.Files = append(cfg.Files, client.ConfigFile{Name: parts[0], Value: enc})
	}

	if err := api.DeviceCreateConfig(device.Name, cfg); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
