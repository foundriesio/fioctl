package targets

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"
	"gopkg.in/yaml.v2"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	showCmd := &cobra.Command{
		Use:   "show <version>",
		Short: "Show details of a specific target.",
		Run:   doShow,
		Args:  cobra.ExactArgs(1),
		Example: `
  # Show details of all Targets with version 42:
  fioctl targets show 42

  # Show a specific Target by name:
  fioctl targets show intel-corei7-64-lmp-42`,
	}
	cmd.AddCommand(showCmd)
	showCmd.PersistentFlags().String("production-tag", "", "Look up target from the production tag")

	showAppCmd := &cobra.Command{
		Use:   "compose-app <version> <app>",
		Short: "Show details of a specific compose app.",
		Run:   doShowComposeApp,
		Args:  cobra.ExactArgs(2),
	}
	showCmd.AddCommand(showAppCmd)
	showAppCmd.Flags().Bool("manifest", false, "Show an app docker manifest")
}

func sortedAppsNames(target client.TufCustom) []string {
	keys := make([]string, 0, len(target.ComposeApps))
	for app := range target.ComposeApps {
		keys = append(keys, app)
	}
	sort.Strings(keys)
	return keys
}

func doShow(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	version := args[0]
	logrus.Debugf("Showing targets for %s %s", factory, version)

	prodTag, _ := cmd.Flags().GetString("production-tag")

	shownCiUrl := false
	sortedTargetNames, hashes, targets := getTargets(factory, prodTag, version)
	for _, targetName := range sortedTargetNames {
		target := targets[targetName]
		hash := hashes[targetName]
		if !shownCiUrl {
			shownCiUrl = true
			fmt.Printf("CI:\thttps://app.foundries.io/factories/%s/targets/%s/\n", factory, target.Version)
		}
		fmt.Println("\n## Target:", targetName)
		fmt.Printf("\tCreated:       %s\n", target.CreatedAt)
		fmt.Printf("\tTags:          %s\n", strings.Join(target.Tags, ","))
		fmt.Printf("\tOSTree Hash:   %s\n", hash)
		if len(target.OrigUri) > 0 {
			parts := strings.Split(target.OrigUri, "/")
			fmt.Printf("\tOrigin Target: %s\n", parts[len(parts)-1])
		}
		fmt.Println()
		fmt.Println("\tSource:")
		if len(target.LmpManifestSha) > 0 {
			fmt.Printf("\t\thttps://source.foundries.io/factories/%s/lmp-manifest.git/commit/?id=%s\n", factory, target.LmpManifestSha)
		}
		if len(target.OverridesSha) > 0 {
			fmt.Printf("\t\thttps://source.foundries.io/factories/%s/meta-subscriber-overrides.git/commit/?id=%s\n", factory, target.OverridesSha)
		}
		if len(target.ContainersSha) > 0 {
			fmt.Printf("\t\thttps://source.foundries.io/factories/%s/containers.git/commit/?id=%s\n", factory, target.ContainersSha)
		}
		fmt.Println()
		t := subcommands.Tabby(1, "APP", "HASH")
		sortedApps := sortedAppsNames(target)
		for _, name := range sortedApps {
			app := target.ComposeApps[name]
			t.AddLine(name, app.Hash())
		}
		t.Print()
	}
}

func doShowComposeApp(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	version := args[0]
	appName := args[1]
	logrus.Debugf("Showing target for %s %s %s", factory, version, appName)

	prodTag, _ := cmd.Flags().GetString("production-tag")

	_, _, targets := getTargets(factory, prodTag, version)
	for name, custom := range targets {
		_, ok := custom.ComposeApps[appName]
		if !ok {
			fmt.Println("ERROR: App not found in target")
			os.Exit(1)
		}
		appInfo, err := api.TargetComposeApp(factory, name, appName)
		subcommands.DieNotNil(err)

		fmt.Println("Version:\n\t", appInfo.Uri)
		if len(appInfo.Error) > 0 {
			fmt.Println("Error:\n\t", appInfo.Error)
		}
		if appInfo.Warnings != nil {
			fmt.Println("Warnings:")
			for _, warn := range appInfo.Warnings {
				fmt.Println("\t", warn)
			}
		}

		if appInfo.Manifest != nil {
			if showManifest, _ := cmd.Flags().GetBool("manifest"); showManifest {
				fmt.Println("\nDocker Manifest:")
				// If a JSON unmarshal worked - YAML marshal will also work
				manifest, _ := yaml.Marshal(appInfo.Manifest)
				fmt.Println(indent(strings.TrimSpace(string(manifest)), "\t"))
			}
		}

		if appInfo.Content.Files != nil {
			fmt.Println("\nFiles:")
			for _, file := range appInfo.Content.Files {
				fmt.Println("\t", file)
			}
		}

		if appInfo.Content.ComposeSpec != nil {
			fmt.Println("\nDocker Compose Template:")
			// If a JSON unmarshal worked - YAML marshal will also work
			composeSpec, _ := yaml.Marshal(appInfo.Content.ComposeSpec)
			fmt.Println(indent(strings.TrimSpace(string(composeSpec)), "\t"))
		}

		// all targets contain the same compose apps, so only print once
		break
	}
}

func getTargets(factory string, prodTag string, version string) ([]string, map[string]string, map[string]client.TufCustom) {
	var targets tuf.Files
	var prodMeta *client.AtsTufTargets

	byName := true
	if _, err := strconv.Atoi(version); err == nil {
		byName = false
	}

	if len(prodTag) > 0 {
		var err error
		prodMeta, err = api.ProdTargetsGet(factory, prodTag, true)
		subcommands.DieNotNil(err)
		targets = make(tuf.Files)
		if !byName {
			for name, target := range prodMeta.Signed.Targets {
				custom, err := api.TargetCustom(target)
				subcommands.DieNotNil(err)
				if custom.Version == version {
					targets[name] = target
				}
			}
		} else {
			targets[version] = prodMeta.Signed.Targets[version]
		}
	} else if !byName {
		var err error
		logrus.Debug("Looking up targets by version")
		targets, err = api.TargetsList(factory, version)
		subcommands.DieNotNil(err)
	} else {
		logrus.Debug("Looking up target by name")
		target, err := api.TargetGet(factory, version)
		subcommands.DieNotNil(err)
		targets = make(tuf.Files)
		targets[version] = *target
	}

	var names []string
	hashes := make(map[string]string)
	matches := make(map[string]client.TufCustom)
	for name, target := range targets {
		custom, err := api.TargetCustom(target)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		if custom.TargetFormat != "OSTREE" {
			logrus.Debugf("Skipping non-ostree target: %v", target)
			continue
		}
		matches[name] = *custom
		hashes[name] = base64.StdEncoding.EncodeToString(target.Hashes["sha256"])
		names = append(names, name)
	}
	if len(matches) == 0 {
		fmt.Println("ERROR: no target found for this version")
		os.Exit(1)
	}
	sort.Strings(names)
	return names, hashes, matches
}

func indent(input string, prefix string) string {
	return prefix + strings.Join(strings.SplitAfter(input, "\n"), prefix)
}
