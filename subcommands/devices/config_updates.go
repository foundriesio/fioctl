package devices

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
)

var (
	updateTags string
	updateApps string
)

// Aktualizr puts all config files into a single lexographically sorted map.
// We have to make sure this file is parsed *after* sota.toml.
const tomlName = "z-50-fioctl.toml"

func init() {
	configUpdatesCmd := &cobra.Command{
		Use:   "updates <device>",
		Short: "Configure aktualizr-lite settings for how updates are applied to a device",
		Run:   doConfigUpdates,
		Long: `View or change configuration parameters used by aktualizr-lite for updating a device.
When run with no options, this command will print out how the device is
currently configured and reporting. Configuration can be updated with commands
like:

  # Make a device start taking updates from Targets tagged with "devel"
  fioctl devices config updates <device> --tags devel

  # Make a device start taking updates from 2 different tags:
  fioctl devices config updates <device> --tags devel,devel-foo

  # Set the docker apps a device will run:
  fioctl devices config updates <device> --apps shellhttpd

  # Set the docker apps and the tag:
  fioctl devices config updates <device> --apps shellhttpd --tags master
`,
	}
	configCmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringVarP(&updateTags, "tags", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().StringVarP(&updateApps, "apps", "", "", "comma,separate,list")
}

func loadSotaConfig(device string) *toml.Tree {
	dcl, err := api.DeviceListConfig(device)
	if err == nil && len(dcl.Configs) > 0 {
		for _, cfgFile := range dcl.Configs[0].Files {
			if cfgFile.Name == tomlName {
				sota, err := toml.Load(cfgFile.Value)
				if err != nil {
					fmt.Println("ERROR - unable to decode toml:", err)
					fmt.Println("      - TOML is:", cfgFile.Value)
					os.Exit(1)
				}
				return sota
			}
		}
	}

	tree, _ := toml.Load("[pacman]")
	return tree
}

// Validate the inputs: Must be alphanumeric, a dash, underscore, or comma
func validateUpdateArgs() {
	pattern := `^[a-zA-Z0-9-_,]+$`
	re := regexp.MustCompile(pattern)
	if len(updateApps) > 0 && !re.MatchString(updateApps) {
		fmt.Println("ERROR: Invalid value for apps:", updateApps)
		fmt.Println("       apps must be ", pattern)
		os.Exit(1)
	}
	if len(updateTags) > 0 && !re.MatchString(updateTags) {
		fmt.Println("ERROR: Invalid value for tags:", updateTags)
		fmt.Println("       apps must be ", pattern)
		os.Exit(1)
	}
}

func doConfigUpdates(cmd *cobra.Command, args []string) {
	logrus.Debug("Configuring device updates")

	// Ensure the device has a public key we can encrypt with
	device, err := api.DeviceGet(args[0])
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}

	sota := loadSotaConfig(device.Name)
	configuredApps := sota.GetDefault("pacman.docker_apps", "").(string)
	configuredTags := sota.GetDefault("pacman.tags", "").(string)

	if len(updateTags) == 0 && len(updateApps) == 0 {
		fmt.Println("= Reporting to server with")
		fmt.Println(" Tags: ", strings.Join(device.Tags, ","))
		fmt.Println(" Apps: ", strings.Join(device.DockerApps, ","))

		fmt.Println("\n= Configured overrides")
		fmt.Println(sota)
		return
	}

	validateUpdateArgs()

	changed := false
	if configuredApps != updateApps {
		fmt.Printf("Changing apps from: [%s] -> [%s]\n", configuredApps, updateApps)
		sota.Set("pacman.docker_apps", updateApps)
		changed = true
	}
	if configuredTags != updateTags {
		fmt.Printf("Changing tags from: [%s] -> [%s]\n", configuredTags, updateTags)
		sota.Set("pacman.tags", updateTags)
		changed = true
	}

	if !changed {
		fmt.Println("ERROR - no changes found. Device is already configured with the specified options.")
		os.Exit(1)
	}

	newToml, err := sota.ToTomlString()
	if err != nil {
		fmt.Println("ERROR: Unable to encode toml: ", err)
		os.Exit(1)
	}

	cfg := client.ConfigCreateRequest{
		Reason: "Override aktualizr-lite update configuration ",
		Files: []client.ConfigFile{
			client.ConfigFile{
				Name:        tomlName,
				Unencrypted: true,
				OnChanged:   []string{"/usr/share/fioconfig/handlers/aktualizr-toml-update"},
				Value:       newToml,
			},
		},
	}
	if err := api.DevicePatchConfig(args[0], cfg); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
