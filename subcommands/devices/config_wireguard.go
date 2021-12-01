package devices

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/subcommands/config"
)

func init() {
	configCmd.AddCommand(&cobra.Command{
		Use:   "wireguard <device> [enable|disable]",
		Short: "Enable or disable wireguard VPN for this device",
		Run:   doConfigWireguard,
		Args:  cobra.RangeArgs(1, 2),
	})
}

type WireguardClientConfig struct {
	Enabled   bool
	Address   string
	PublicKey string
}

func (w WireguardClientConfig) Marshall() string {
	buff := "address=" + w.Address + "\npubkey=" + w.PublicKey
	if !w.Enabled {
		buff += "\nenabled=0"
	}
	return buff
}

func (w *WireguardClientConfig) Unmarshall(configVal string) {
	w.Enabled = true
	for _, line := range strings.Split(configVal, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		if k == "address" {
			w.Address = strings.TrimSpace(parts[1])
		} else if k == "enabled" {
			w.Enabled = parts[1] != "0"
		} else if k == "pubkey" {
			w.PublicKey = strings.TrimSpace(parts[1])
		} else {
			fmt.Println("ERROR: Unexpected client config key: ", k)
			os.Exit(1)
		}
	}
}

func loadWireguardClientConfig(factory, device string) WireguardClientConfig {
	dcl, err := api.DeviceListConfig(factory, device)
	wcc := WireguardClientConfig{}
	if err != nil {
		return wcc
	}
	if len(dcl.Configs) > 0 {
		for _, cfgFile := range dcl.Configs[0].Files {
			if cfgFile.Name == "wireguard-client" {
				wcc.Unmarshall(cfgFile.Value)
				break
			}
		}
	}
	return wcc
}

// Convert an IP into an uint32 so we can easily compare
func ipToUint32(ipaddr string) (uint32, error) {
	ip := net.ParseIP(ipaddr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address: %s", ipaddr)
	}
	return binary.BigEndian.Uint32(ip.To4()), nil
}

// Create a dictionary of device VPN addresses in the factory
func factoryIps(factory string) map[uint32]bool {
	ips := make(map[uint32]bool)
	ipList, err := api.GetWireGuardIps(factory)
	subcommands.DieNotNil(err)
	for _, item := range ipList {
		ip, err := ipToUint32(item.Ip)
		if err != nil {
			logrus.Errorf("Unable to compute VPN Address for %s - %s", item.Name, item.Ip)
		} else {
			ips[ip] = true
		}
	}
	return ips
}

func findVpnAddress(factory string) string {
	wsc := config.LoadWireguardServerConfig(factory, api)
	if len(wsc.VpnAddress) == 0 || !wsc.Enabled {
		fmt.Println("ERROR: A wireguard server has not been configured for this factory")
		os.Exit(1)
	}
	logrus.Debugf("VPN server address is: %s", wsc.VpnAddress)
	serverIp, err := ipToUint32(wsc.VpnAddress)
	if err != nil {
		fmt.Println("ERROR: Wireguard server has an invalid IP Address: ", wsc.VpnAddress)
		os.Exit(1)
	}

	ips := factoryIps(factory)
	for ip := serverIp + 1; ip < serverIp+10000; ip++ {
		if _, ok := ips[ip]; !ok {
			logrus.Debugf("Found unique ip: %d", ip)
			return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
		}
	}

	fmt.Println("ERROR: Unable to find unique IP address for VPN")
	os.Exit(1)
	return ""
}

func doConfigWireguard(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debug("Configuring wireguard")

	// Ensure the device has a public key we can encrypt with
	wcc := loadWireguardClientConfig(factory, args[0])
	if len(args) == 1 {
		fmt.Println("Enabled:", wcc.Enabled)
		if len(wcc.Address) > 0 {
			fmt.Println("Address:", wcc.Address)
		}
		if len(wcc.PublicKey) > 0 {
			fmt.Println("Public Key:", wcc.PublicKey)
		}
		os.Exit(0)
	} else if args[1] != "enable" && args[1] != "disable" {
		fmt.Printf("Invalid argument: '%s'. Must be 'enabled' or 'disabled'\n", args[1])
		os.Exit(0)
	}

	cfg := client.ConfigCreateRequest{
		Reason: "Set Wireguard configuration - " + args[1],
		Files: []client.ConfigFile{
			{
				Name:        "wireguard-client",
				Unencrypted: true,
				OnChanged:   []string{"/usr/share/fioconfig/handlers/factory-config-vpn"},
			},
		},
	}

	if args[1] == "enable" {
		if len(wcc.PublicKey) == 0 {
			fmt.Println("ERROR: Device has no public key for VPN")
			os.Exit(1)
		}
		wcc.Enabled = true
		if len(wcc.Address) == 0 {
			fmt.Println("Finding a unique VPN address ...")
			wcc.Address = findVpnAddress(factory)
		}
	} else {
		wcc.Enabled = false
	}
	cfg.Files[0].Value = wcc.Marshall()
	subcommands.DieNotNil(api.DevicePatchConfig(factory, args[0], cfg, false))
}
