package config

import (
	"fmt"
	"strings"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rotateCmd := &cobra.Command{
		Use:   "rotate-certs",
		Short: "Rotate device x509 keypairs in this group used to connect to the device gateway",
		Run:   doCertRotate,
		Long: `This command will send a fioconfig change to a device to instruct it to perform
a certificate rotation using the EST server configured with "fioctl keys est".

This command will only work for devices running LmP version 90 and later.`,
	}
	cmd.AddCommand(rotateCmd)
	rotateCmd.Flags().StringP("group", "g", "", "Device group to use")
	rotateCmd.Flags().StringP("est-resource", "e", "/.well-known/est", "Path to the EST resource on your server")
	rotateCmd.Flags().IntP("est-port", "p", 8443, "EST server port")
	rotateCmd.Flags().StringP("reason", "r", "", "reason for changing the cert")
	rotateCmd.Flags().StringP("hsm-pkey-ids", "", "01,07", "Available PKCS11 slot IDs for the private key")
	rotateCmd.Flags().StringP("hsm-cert-ids", "", "03,09", "Available PKCS11 slot IDs for the client certificate")
	rotateCmd.Flags().StringP("server-name", "", "", "EST server name when not using the Foundries.io managed server. e.g. est.example.com")
	rotateCmd.Flags().BoolP("dryrun", "", false, "Show what the fioconfig entry will be and exit")
	_ = cmd.MarkFlagRequired("reason")
	_ = cmd.MarkFlagRequired("group")
}

func doCertRotate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	estResource, _ := cmd.Flags().GetString("est-resource")
	estPort, _ := cmd.Flags().GetInt("est-port")
	group, _ := cmd.Flags().GetString("group")
	keyIds, _ := cmd.Flags().GetString("hsm-pkey-ids")
	certIds, _ := cmd.Flags().GetString("hsm-cert-ids")
	reason, _ := cmd.Flags().GetString("reason")
	serverName, _ := cmd.Flags().GetString("server-name")
	dryRun, _ := cmd.Flags().GetBool("dryrun")

	if estResource[0] != '/' {
		estResource = "/" + estResource
	}

	var url string
	if len(serverName) > 0 {
		url = fmt.Sprintf("https://%s:%d%s", serverName, estPort, estResource)
	} else {
		var err error
		url, err = api.FactoryEstUrl(factory, estPort, estResource)
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
	subcommands.DieNotNil(api.GroupPatchConfig(factory, group, ccr, false))
}
