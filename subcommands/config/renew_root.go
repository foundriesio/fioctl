package config

import (
	"errors"
	"fmt"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	renewCmd := &cobra.Command{
		Use:   "renew-root",
		Short: "Renew a Factory root CA on the devices (in this group) used to verify the device gateway TLS certificate",
		Run:   doRenewRoot,
		Long: `This command will send a fioconfig change to a device to instruct it to perform
a root CA renewal using the EST server configured with "fioctl keys est".
If there is no configured EST server, it will instruct the device to renew a root CA from the device gateway.

This command will only work for devices running LmP version 95 and later.`,
	}
	cmd.AddCommand(renewCmd)
	renewCmd.Flags().StringP("group", "g", "", "Device group to use")
	renewCmd.Flags().StringP("est-resource", "e", "/.well-known/cacerts", "The path the to EST resource on your server")
	renewCmd.Flags().IntP("est-port", "p", 8443, "The EST server port")
	renewCmd.Flags().StringP("reason", "r", "", "The reason for triggering the root CA renewal")
	renewCmd.Flags().StringP("server-name", "", "", "EST server name when not using the Foundries managed server. e.g. est.example.com")
	renewCmd.Flags().BoolP("dryrun", "", false, "Show what the fioconfig entry will be and exit")
	_ = renewCmd.MarkFlagRequired("reason")
}

func doRenewRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	estResource, _ := cmd.Flags().GetString("est-resource")
	estPort, _ := cmd.Flags().GetInt("est-port")
	group, _ := cmd.Flags().GetString("group")
	reason, _ := cmd.Flags().GetString("reason")
	serverName, _ := cmd.Flags().GetString("server-name")
	dryRun, _ := cmd.Flags().GetBool("dryrun")

	if estResource[0] != '/' {
		estResource = "/" + estResource
	}

	if len(group) > 0 {
		logrus.Debugf("Renewing device root CA for devices in group %s", group)
	} else {
		logrus.Debug("Renewing device root CA for all factory devices")
	}

	certs, err := api.FactoryGetCA(factory)
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
	if len(group) > 0 {
		subcommands.DieNotNil(api.GroupPatchConfig(factory, group, ccr, false))
	} else {
		subcommands.DieNotNil(api.FactoryPatchConfig(factory, ccr, false))
	}
}
