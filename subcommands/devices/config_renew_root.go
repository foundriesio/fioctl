package devices

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd := &cobra.Command{
		Use:   "renew-root <device>",
		Short: "Renew a Factory root CA on the device used to verify the device gateway TLS certificate",
		Args:  cobra.ExactArgs(1),
		Run:   doConfigRenewRoot,
		Long: `This command will send a fioconfig change to a device to instruct it to perform
a root CA renewal using the EST server configured with "fioctl keys est".
If there is no configured EST server, it will instruct the device to renew a root CA from the device gateway.

This command will only work for devices running LmP version 95 and later.`,
	}
	cmd.Flags().StringP("est-resource", "e", "/.well-known/cacerts", "The path the to EST resource on your server")
	cmd.Flags().IntP("est-port", "p", 8443, "The EST server port")
	cmd.Flags().StringP("reason", "r", "", "The reason for triggering the root CA renewal")
	cmd.Flags().StringP("server-name", "", "", "EST server name when not using the Foundries managed server. e.g. est.example.com")
	cmd.Flags().BoolP("dryrun", "", false, "Show what the fioconfig entry will be and exit")
	configCmd.AddCommand(cmd)
	_ = cmd.MarkFlagRequired("reason")
}

func doConfigRenewRoot(cmd *cobra.Command, args []string) {
	name := args[0]
	estResource, _ := cmd.Flags().GetString("est-resource")
	estPort, _ := cmd.Flags().GetInt("est-port")
	reason, _ := cmd.Flags().GetString("reason")
	serverName, _ := cmd.Flags().GetString("server-name")
	dryRun, _ := cmd.Flags().GetBool("dryrun")

	if estResource[0] != '/' {
		estResource = "/" + estResource
	}

	logrus.Debugf("Renewing device root CA for %s", name)

	// Quick sanity check for device
	d := getDevice(cmd, name)
	certs, err := api.FactoryGetCA(d.Factory)
	subcommands.DieNotNil(err)
	if len(certs.RootRenewalCorrelationId) == 0 {
		subcommands.DieNotNil(errors.New("There is no Factory root renewal. Use 'fioctl keys ca renewal' to start it."))
	}

	var url string
	if len(serverName) > 0 {
		url = fmt.Sprintf("https://%s:%d%s", serverName, estPort, estResource)
	} else {
		url, err = certs.GetEstUrl(estPort, estResource, true)
		subcommands.DieNotNil(err)
	}
	logrus.Debugf("Using EST server: %s", url)

	opts := subcommands.RenewRootOptions{
		Reason:        reason,
		CorrelationId: certs.RootRenewalCorrelationId,
		EstServer:     url,
	}

	ccr := opts.AsConfig()
	if dryRun {
		fmt.Println("Config file would be:")
		fmt.Println(ccr.Files[0].Value)
		return
	}
	subcommands.DieNotNil(d.Api.PatchConfig(ccr, false))
}
