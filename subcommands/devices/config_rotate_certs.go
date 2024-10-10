package devices

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rotate-certs <device>",
		Short: "Rotate a device's x509 keypair used to connect to the device gateway",
		Args:  cobra.ExactArgs(1),
		Run:   doConfigRotate,
		Long: `This command will send a fioconfig change to a device, instructing it to perform
a certificate rotation using the EST server configured with "fioctl keys est".

This command will only work for devices running LmP version 90 and later.`,
	}
	cmd.Flags().StringP("est-resource", "e", "/.well-known/est", "The path the to EST resource on your server")
	cmd.Flags().IntP("est-port", "p", 8443, "The EST server port")
	cmd.Flags().StringP("reason", "r", "", "The reason for changing the cert")
	cmd.Flags().StringP("hsm-pkey-ids", "", "01,07", "Available PKCS11 slot IDs for the private key")
	cmd.Flags().StringP("hsm-cert-ids", "", "03,09", "Available PKCS11 slot IDs for the client certificate")
	cmd.Flags().StringP("server-name", "", "", "EST server name when not using the Foundries.io managed server. e.g. est.example.com")
	cmd.Flags().BoolP("dryrun", "", false, "Show what the fioconfig entry will be and exit")
	configCmd.AddCommand(cmd)
	_ = cmd.MarkFlagRequired("reason")
}

func doConfigRotate(cmd *cobra.Command, args []string) {
	name := args[0]
	estResource, _ := cmd.Flags().GetString("est-resource")
	estPort, _ := cmd.Flags().GetInt("est-port")
	keyIds, _ := cmd.Flags().GetString("hsm-pkey-ids")
	certIds, _ := cmd.Flags().GetString("hsm-cert-ids")
	reason, _ := cmd.Flags().GetString("reason")
	serverName, _ := cmd.Flags().GetString("server-name")
	dryRun, _ := cmd.Flags().GetBool("dryrun")

	if estResource[0] != '/' {
		estResource = "/" + estResource
	}

	logrus.Debugf("Rotating device certs for %s", name)

	// Quick sanity check for device
	d := getDevice(cmd, name)

	var err error
	var url string
	if len(serverName) > 0 {
		url = fmt.Sprintf("https://%s:%d%s", serverName, estPort, estResource)
	} else {
		url, err = api.FactoryEstUrl(d.Factory, estPort, estResource)
		subcommands.DieNotNil(err)
	}
	logrus.Debugf("Using EST server: %s", url)

	opts := subcommands.RotateCertOptions{
		Reason:    reason,
		EstServer: url,
		PkeyIds:   strings.Split(keyIds, ","),
		CertIds:   strings.Split(certIds, ","),
	}

	ccr := opts.AsConfig()
	if dryRun {
		fmt.Println("Config file would be:")
		fmt.Println(ccr.Files[0].Value)
		return
	}
	subcommands.DieNotNil(d.Api.PatchConfig(ccr, false))
}
