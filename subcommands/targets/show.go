package targets

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	}
	cmd.AddCommand(showCmd)

	showAppCmd := &cobra.Command{
		Use:   "compose-app <version> <app>",
		Short: "Show details of a specific compose app.",
		Run:   doShowComposeApp,
		Args:  cobra.ExactArgs(2),
	}
	showCmd.AddCommand(showAppCmd)
	showAppCmd.Flags().Bool("manifest", false, "Show an app docker manifest")
}

func doShow(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	version := args[0]
	logrus.Debugf("Showing targets for %s %s", factory, version)

	var tags []string
	var apps map[string]client.DockerApp
	var composeApps map[string]client.ComposeApp
	containersSha := ""
	manifestSha := ""
	overridesSha := ""
	hashes, targets := getTargets(factory, version)
	for _, custom := range targets {
		if len(custom.ContainersSha) > 0 {
			if len(containersSha) > 0 && containersSha != custom.ContainersSha {
				fmt.Println("ERROR: Git hashes for containers.git does not match across platforms")
				os.Exit(1)
			}
			containersSha = custom.ContainersSha
		}
		if len(custom.LmpManifestSha) > 0 {
			if len(manifestSha) > 0 && manifestSha != custom.LmpManifestSha {
				fmt.Println("ERROR: Git hashes for lmp-manifest.git does not match across platforms")
				os.Exit(1)
			}
			manifestSha = custom.LmpManifestSha
		}
		if len(custom.OverridesSha) > 0 {
			if len(overridesSha) > 0 && overridesSha != custom.OverridesSha {
				fmt.Println("ERROR: Git hashes for meta-subscriber-overrides.git does not match across platforms")
				os.Exit(1)
			}
			overridesSha = custom.OverridesSha
		}
		apps = custom.DockerApps
		composeApps = custom.ComposeApps
		tags = custom.Tags
	}

	fmt.Printf("Tags:\t%s\n", strings.Join(tags, ","))

	fmt.Printf("CI:\thttps://ci.foundries.io/projects/%s/lmp/builds/%s/\n", factory, version)

	fmt.Println("Source:")
	if len(manifestSha) > 0 {
		fmt.Printf("\thttps://source.foundries.io/factories/%s/lmp-manifest.git/commit/?id=%s\n", factory, manifestSha)
	}
	if len(overridesSha) > 0 {
		fmt.Printf("\thttps://source.foundries.io/factories/%s/meta-subscriber-overrides.git/commit/?id=%s\n", factory, overridesSha)
	}
	if len(containersSha) > 0 {
		fmt.Printf("\thttps://source.foundries.io/factories/%s/containers.git/commit/?id=%s\n", factory, containersSha)
	}
	fmt.Println("")

	t := tabby.New()
	t.AddHeader("TARGET NAME", "OSTREE HASH - SHA256")
	for name, val := range hashes {
		t.AddLine(name, val)
	}
	t.Print()

	fmt.Println()

	if len(apps) > 0 {
		t = tabby.New()
		t.AddHeader("DOCKER APP", "VERSION")
		for name, app := range apps {
			if len(app.FileName) > 0 {
				t.AddLine(name, app.FileName)
			}
			if len(app.Uri) > 0 {
				t.AddLine(name, app.Uri)
			}
		}
		t.Print()
	}
	if len(composeApps) > 0 {
		if len(apps) > 0 {
			fmt.Println()
		}
		t = tabby.New()
		t.AddHeader("COMPOSE APP", "VERSION")
		for name, app := range composeApps {
			t.AddLine(name, app.Uri)
		}
		t.Print()
	}
}

func doShowComposeApp(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	version := args[0]
	appName := args[1]
	logrus.Debugf("Showing target for %s %s %s", factory, version, appName)

	_, targets := getTargets(factory, version)
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

func getTargets(factory string, version string) (map[string]string, map[string]client.TufCustom) {
	targets, err := api.TargetsList(factory, version)
	subcommands.DieNotNil(err)

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
	}
	if len(matches) == 0 {
		fmt.Println("ERROR: no OSTREE target found for this version")
		os.Exit(1)
	}
	return hashes, matches
}

func indent(input string, prefix string) string {
	return prefix + strings.Join(strings.SplitAfter(input, "\n"), prefix)
}
