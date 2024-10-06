package keys

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	X509 "github.com/foundriesio/fioctl/x509"
)

var (
	hsmModule     string
	hsmPin        string
	hsmTokenLabel string
)

func addStandardHsmFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&hsmModule, "hsm-module", "", "", "Load a root CA key from a PKCS#11 compatible HSM using this module")
	cmd.Flags().StringVarP(&hsmPin, "hsm-pin", "", "", "The PKCS#11 PIN to log into the HSM")
	cmd.Flags().StringVarP(&hsmTokenLabel, "hsm-token-label", "", "", "The label of the HSM token containing the root CA key")
}

func validateStandardHsmArgs(hsmModule, hsmPin, hsmTokenLabel string) (*X509.HsmInfo, error) {
	return validateHsmArgs(hsmModule, hsmPin, hsmTokenLabel, "--hsm-module", "--hsm-pin", "--hsm-token-label")
}

func validateHsmArgs(hsmModule, hsmPin, hsmTokenLabel, moduleArg, pinArg, tokenArg string) (*X509.HsmInfo, error) {
	if len(hsmModule) > 0 {
		if len(hsmPin) == 0 {
			return nil, fmt.Errorf("%s is required with %s", pinArg, moduleArg)
		}
		if len(hsmTokenLabel) == 0 {
			return nil, fmt.Errorf("%s is required with %s", tokenArg, moduleArg)
		}
		return &X509.HsmInfo{Module: hsmModule, Pin: hsmPin, TokenLabel: hsmTokenLabel}, nil
	}
	return nil, nil
}

func parseCertList(pemData string) (certs []*x509.Certificate) {
	for len(pemData) > 0 {
		block, remaining := pem.Decode([]byte(pemData))
		if block == nil {
			// could be excessive whitespace
			if pemData = strings.TrimSpace(string(remaining)); len(pemData) == len(remaining) {
				fmt.Println("Failed to parse remaining certificates: invalid PEM data")
				break
			}
			continue
		}
		pemData = string(remaining)
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			fmt.Println("Failed to parse certificate:" + err.Error())
			continue
		}
		certs = append(certs, c)
	}
	return
}
