package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var wireguardDisable bool

func init() {
	wireguardCmd := &cobra.Command{
		Use:   "wireguard",
		Short: "Show current wireguard server config for factory",
		Run:   doWireguard,
		Args:  cobra.MinimumNArgs(0),
	}
	cmd.AddCommand(wireguardCmd)
	wireguardCmd.Flags().BoolVarP(&wireguardDisable, "disable", "", false, "Disable VPN access for all devices")
}

type WireguardServerConfig struct {
	Enabled    bool
	Endpoint   string
	VpnAddress string
	PublicKey  string
}

func (w WireguardServerConfig) Marshall() string {
	buff := "endpoint=" + w.Endpoint + "\nserver_address=" + w.VpnAddress + "\npubkey=" + w.PublicKey
	if !w.Enabled {
		buff += "\nenabled=0"
	}
	return buff
}

func (w *WireguardServerConfig) Unmarshall(configVal string) {
	w.Enabled = true
	for _, line := range strings.Split(configVal, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "endpoint" {
			w.Endpoint = v
		} else if k == "server_address" {
			w.VpnAddress = v
		} else if k == "pubkey" {
			w.PublicKey = v
		} else if k == "enabled" {
			w.Enabled = v != "0"
		} else {
			fmt.Println("ERROR: Unexpected client config key: ", k)
			os.Exit(1)
		}
	}
}

func LoadWireguardServerConfig(factory string, api *client.Api) WireguardServerConfig {
	dcl, err := api.FactoryListConfig(factory)
	subcommands.DieNotNil(err)

	wsc := WireguardServerConfig{}
	if len(dcl.Configs) > 0 {
		for _, cfgFile := range dcl.Configs[0].Files {
			if cfgFile.Name == "wireguard-server" {
				logrus.Debugf("Found existing server config: %s", cfgFile.Value)
				wsc.Unmarshall(cfgFile.Value)
				break
			}
		}
	}
	return wsc
}

func doWireguard(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Creating new config for %s", factory)

	wsc := LoadWireguardServerConfig(factory, api)

	if wireguardDisable {
		wsc.Enabled = false
		cfg := client.ConfigCreateRequest{
			Reason: "Disable Wireguard configuration",
			Files: []client.ConfigFile{
				client.ConfigFile{
					Name:        "wireguard-server",
					Unencrypted: true,
					OnChanged:   []string{"/usr/share/fioconfig/handlers/factory-config-vpn"},
				},
			},
		}
		cfg.Files[0].Value = wsc.Marshall()

		subcommands.DieNotNil(api.FactoryPatchConfig(factory, cfg))
	} else {
		fmt.Println("Enabled:", wsc.Enabled)
		if len(wsc.Endpoint) > 0 {
			fmt.Println("Endpoint:", wsc.Endpoint)
		}
		if len(wsc.VpnAddress) > 0 {
			fmt.Println("Address:", wsc.VpnAddress)
		}
		if len(wsc.PublicKey) > 0 {
			fmt.Println("Public Key:", wsc.PublicKey)
		}
	}

}
