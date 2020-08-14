package devices

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var (
	dryRun      bool
	force       bool
	updateTags  string
	updateApps  string
	composeApps bool
	composeDir  string
)

// Aktualizr puts all config files into a single lexographically sorted map.
// We have to make sure this file is parsed *after* sota.toml.
const tomlName = "z-50-fioctl.toml"

func init() {
	configUpdatesCmd := &cobra.Command{
		Use:   "updates <device>",
		Short: "Configure aktualizr-lite settings for how updates are applied to a device",
		Run:   doConfigUpdates,
		Args:  cobra.ExactArgs(1),
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

  # Move device from old docker-apps to compose-apps:
  fioctl devices config updates <device> --compose-apps
`,
	}
	configCmd.AddCommand(configUpdatesCmd)
	configUpdatesCmd.Flags().StringVarP(&updateTags, "tags", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().StringVarP(&updateApps, "apps", "", "", "comma,separate,list")
	configUpdatesCmd.Flags().BoolVarP(&composeApps, "compose-apps", "", false, "Migrate device from docker-apps to compose-apps")
	configUpdatesCmd.Flags().StringVarP(&composeDir, "compose-dir", "", "/var/sota/compose", "The directory to install compose apps in")
	configUpdatesCmd.Flags().BoolVarP(&dryRun, "dryrun", "", false, "Only show what would be changed")
	configUpdatesCmd.Flags().BoolVarP(&force, "force", "", false, "Apply change even if validation rejects it")
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

func exitIfNotForce() {
	if !force {
		os.Exit(1)
	}
}

func versionInt(ver string) int {
	verI, err := strconv.Atoi(ver)
	if err != nil {
		panic(fmt.Sprintf("Invalid version: %v. Error: %s", ver, err))
	}
	return verI
}

func appsMatch(targetApps, proposedApps []string) bool {
	for _, a := range proposedApps {
		match := false
		for _, targetApp := range targetApps {
			if a == targetApp {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// Validate the inputs: Must be alphanumeric, a dash, underscore, or comma
func validateUpdateArgs(device *client.Device) {
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

	tags := device.Tags
	if len(updateTags) > 0 {
		tags = strings.Split(updateTags, ",")
	}
	apps := device.DockerApps
	if len(updateApps) > 0 {
		apps = strings.Split(updateApps, ",")
	}

	targets, err := api.TargetsList(device.Factory)
	if err != nil {
		fmt.Println("ERROR:", err)
		exitIfNotForce()
	}

	curTarget := targets.Signed.Targets[device.TargetName]
	custom, err := api.TargetCustom(curTarget)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		exitIfNotForce()
	}
	curVer := versionInt(custom.Version)

	// See if the tag exists
	// if it does, is the latest version going to result in Downgrade?
	// if that's all okay, do the apps exist in the Target its destined for?
	found := false
	upgrade := false
	appMismatch := []string{}
	latest := 0
	for _, target := range targets.Signed.Targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		if custom.TargetFormat != "OSTREE" {
			logrus.Debugf("Skipping non-ostree target: %v", target)
			continue
		}
		if subcommands.IntersectionInSlices(tags, custom.Tags) {
			found = true
			ver := versionInt(custom.Version)
			if ver >= curVer {
				upgrade = true
			}
			if ver > latest {
				latest = ver
				if upgrade {
					targetApps := []string{}
					for a := range custom.ComposeApps {
						targetApps = append(targetApps, a)
					}
					if !appsMatch(targetApps, apps) {
						appMismatch = targetApps
					}
				}
			}
		}
	}

	if !found {
		fmt.Printf("ERROR: Given tags %s are not present in targets.json\n", tags)
		exitIfNotForce()
	}
	if !upgrade {
		fmt.Printf("ERROR: Given tags %s appear to result in a downgrade from version %d to %d\n",
			tags, curVer, latest)
		exitIfNotForce()
	}
	if len(apps) > 0 && len(appMismatch) > 0 {
		fmt.Printf("ERROR: Given apps %s not present in Target version %d device would update to: %s\n",
			apps, latest, appMismatch)
		exitIfNotForce()
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
	configuredMgr := sota.GetDefault("pacman.packagemanager", "").(string)

	if len(updateTags) == 0 && len(updateApps) == 0 && !composeApps {
		fmt.Println("= Reporting to server with")
		fmt.Println(" Tags: ", strings.Join(device.Tags, ","))
		fmt.Println(" Apps: ", strings.Join(device.DockerApps, ","))

		fmt.Println("\n= Configured overrides")
		fmt.Println(sota)
		return
	}

	validateUpdateArgs(device)

	changed := false
	if len(updateApps) > 0 && configuredApps != updateApps {
		if strings.TrimSpace(updateApps) == "," {
			updateApps = ""
		}
		fmt.Printf("Changing apps from: [%s] -> [%s]\n", configuredApps, updateApps)
		sota.Set("pacman.docker_apps", updateApps)
		sota.Set("pacman.compose_apps", updateApps)
		changed = true
	}
	if len(updateTags) > 0 && configuredTags != updateTags {
		if strings.TrimSpace(updateTags) == "," {
			updateTags = ""
		}
		fmt.Printf("Changing tags from: [%s] -> [%s]\n", configuredTags, updateTags)
		sota.Set("pacman.tags", updateTags)
		changed = true
	}
	if composeApps && configuredMgr != "ostree+compose_apps" {
		fmt.Printf("Changing packagemanger to %s\n", "ostree+compose_apps")
		sota.Set("pacman.type", "ostree+compose_apps")
		sota.Set("pacman.compose_apps_root", composeDir)
		// the device might be running DockerApps that were set in /var/sota/sota.toml
		// by lmp-device-register, so fallback to what its reporting if we don't find
		// override values set:
		sota.Set("pacman.compose_apps", sota.GetDefault("pacman.docker_apps", device.DockerApps))
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
	if dryRun {
		fmt.Println(newToml)
		return
	}
	if err := api.DevicePatchConfig(args[0], cfg); err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
}
