package targets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"
	yaml "gopkg.in/yaml.v2"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

type formats map[string]string

var sbomFormats formats

func init() {
	showCmd := &cobra.Command{
		Use:   "show <version>",
		Short: "Show details of a specific Target.",
		Run:   doShow,
		Args:  cobra.ExactArgs(1),
		Example: `
  # Show details of all Targets with version 42:
  fioctl targets show 42

  # Show specific Target by name:
  fioctl targets show intel-corei7-64-lmp-42`,
	}
	cmd.AddCommand(showCmd)
	showCmd.PersistentFlags().String("production-tag", "", "Look up Target from the production tag")

	showAppCmd := &cobra.Command{
		Use:   "compose-app <version> <app>",
		Short: "Show details of a specific compose app.",
		Run:   doShowComposeApp,
		Args:  cobra.ExactArgs(2),
	}
	showCmd.AddCommand(showAppCmd)
	showAppCmd.Flags().Bool("manifest", false, "Show an app docker manifest")

	sbomCmd := &cobra.Command{
		Use:   "sboms <version> [<build/run> [<artifact>]] ",
		Short: "Show SBOMs for a specific target.",
		Run:   doShowSboms,
		Args:  cobra.RangeArgs(1, 3),
		Example: `
  # Show all SBOM files for Target version 42:
  fioctl targets show sboms 42

  # Show a subset of the SBOMS for the Target. In this case, the 32-bit Arm
  # container SBOMS:
  fioctl targets show sboms 42 41/build-armhf
 
  # Show overview of a specific SBOM:
  fioctl targets show sboms 42 41/build-armhf alpine:latest/arm.spdx.json
  
  # Show overview of a specific SBOM as CSV:
  fioctl targets show sboms --format csv 42 41/build-armhf alpine:latest/arm.spdx.json
  
  # Download all SBOMS for a Target to /tmp:
  fioctl targets show sboms 42 --download /tmp
  
  # Download a filtered list of SBOMs to /tmp:
  fioctl targets show sboms 42 41/build-armhf alpine:latest/arm.spdx.json --download /tmp
  
  # Download a specific SBOM as cyclonedx:
  fioctl targets show sboms 42 41/build-armhf --download /tmp --format cyclonedx

  # Download all SBOMS for a Target to /tmp as CSV:
  fioctl targets show sboms 42 --download /tmp --format csv`,
	}

	sbomFormats = make(formats, 4)
	sbomFormats["table"] = "table"
	sbomFormats["spdx"] = "application/spdx.json"
	sbomFormats["cyclonedx"] = "application/cyclone.json"
	sbomFormats["csv"] = "text/csv"
	allowed := "table, spdx, cyclonedx, or csv"

	showCmd.AddCommand(sbomCmd)
	sbomCmd.Flags().String("format", "table", "The format to download/display. Must be one of "+allowed)
	sbomCmd.Flags().String("download", "", "Download SBOM(s) to a directory")
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
	logrus.Debugf("Showing Targets for %s %s", factory, version)

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
		if len(target.LmpVer) > 0 {
			fmt.Printf("\tLmP Version:   %s\n", target.LmpVer)
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
	logrus.Debugf("Showing Target for %s %s %s", factory, version, appName)

	prodTag, _ := cmd.Flags().GetString("production-tag")

	_, _, targets := getTargets(factory, prodTag, version)
	for name, custom := range targets {
		_, ok := custom.ComposeApps[appName]
		if !ok {
			fmt.Println("ERROR: App not found in Target")
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
		logrus.Debug("Looking up Targets by version")
		targets, err = api.TargetsList(factory, version)
		subcommands.DieNotNil(err)
	} else {
		logrus.Debug("Looking up Target by name")
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
			logrus.Debugf("Skipping non-ostree Target: %v", target)
			continue
		}
		matches[name] = *custom
		hashes[name] = base64.StdEncoding.EncodeToString(target.Hashes["sha256"])
		names = append(names, name)
	}
	if len(matches) == 0 {
		fmt.Println("ERROR: no Target found for version")
		os.Exit(1)
	}
	sort.Strings(names)
	return names, hashes, matches
}

func indent(input string, prefix string) string {
	return prefix + strings.Join(strings.SplitAfter(input, "\n"), prefix)
}

func getSbomFormat(formatStr string) string {
	format, ok := sbomFormats[formatStr]
	if !ok {
		fmt.Println(sbomFormats)
		subcommands.DieNotNil(fmt.Errorf("Unknown --format: %s", formatStr))
	}
	return format
}

func doShowSboms(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	version := args[0]
	logrus.Debugf("Showing sboms for %s %s", factory, version)

	filter := ""
	if len(args) == 2 {
		filter = args[1]
	}

	prodTag, _ := cmd.Flags().GetString("production-tag")
	formatStr, _ := cmd.Flags().GetString("format")
	downloadPath, _ := cmd.Flags().GetString("download")

	format := getSbomFormat(formatStr)
	name := version
	if factory != "lmp" {
		name = getSbomTargetName(factory, prodTag, version)
	}

	if len(downloadPath) > 0 {
		doDownloadSboms(factory, name, downloadPath, format, args)
		return
	}

	if len(args) == 3 {
		path := fmt.Sprintf("%s/%s", args[1], args[2])
		displaySbom(factory, name, path, format)
		return
	}

	sboms, err := api.TargetSboms(factory, name)
	subcommands.DieNotNil(err)
	t := tabby.New()
	t.AddHeader("BUILD/RUN", "BOM ARTIFACT")
	for _, sbom := range sboms {
		buildRun := sbom.CiBuild + "/" + sbom.CiRun
		if len(filter) == 0 || strings.HasPrefix(buildRun, filter) {
			t.AddLine(buildRun, sbom.Artifact)
		}
	}
	t.Print()
}

func getSbomTargetName(factory, prodTag, version string) string {
	_, _, targets := getTargets(factory, prodTag, version)
	for name := range targets {
		return name
	}
	subcommands.DieNotNil(fmt.Errorf("Unable to find Target for version: %s", version))
	return "" // Make compiler happy
}

func displaySbom(factory, targetName, path, format string) {
	contentType := format
	if format == "table" {
		contentType = "application/spdx.json"
	}
	data, err := api.SbomDownload(factory, targetName, path, contentType)
	subcommands.DieNotNil(err)
	if format == "table" {
		// special handling for default
		var doc client.SpdxDocument
		subcommands.DieNotNil(json.Unmarshal(data, &doc))
		t := tabby.New()
		t.AddHeader("PACKAGE", "VERSION", "LICENSE")
		for _, pkg := range doc.Packages {
			t.AddLine(pkg.Name, pkg.VersionInfo, pkg.License())
		}
		t.Print()
	} else {
		os.Stdout.Write(data)
	}
}

func doDownloadSboms(factory, targetName, downloadPath, format string, args []string) {
	st, err := os.Stat(downloadPath)
	subcommands.DieNotNil(err)
	if !st.IsDir() {
		subcommands.DieNotNil(fmt.Errorf("download path is not a directory: %s", downloadPath))
	}

	if format == "table" {
		format = "application/spdx.json"
	}

	filter := ""
	if len(args) == 2 {
		filter = args[1]
	} else if len(args) == 3 {
		filter = fmt.Sprintf("%s/%s", args[1], args[2])
	}

	prefixMsg := "Downloading"
	if format != "application/spdx.json" {
		prefixMsg = "Converting"
	}

	sboms, err := api.TargetSboms(factory, targetName)
	subcommands.DieNotNil(err)
	for _, sbom := range sboms {
		buildRun := sbom.CiBuild + "/" + sbom.CiRun + "/" + sbom.Artifact
		if len(filter) == 0 || strings.HasPrefix(buildRun, filter) {
			dst := filepath.Join(downloadPath, sbom.CiBuild, sbom.CiRun, sbom.Artifact)
			// dst will have .spdx.json or .spdx.tar.zst - determine a better extension by content-type:
			parts := strings.SplitN(format, "/", 2)
			extension := "." + parts[1]
			dst = strings.Replace(dst, ".spdx.json", extension, 1)
			dst = strings.Replace(dst, ".spdx.tar.zst", extension, 1)
			subcommands.DieNotNil(os.MkdirAll(filepath.Dir(dst), st.Mode()))
			fmt.Printf("%s %s/%s/%s\n |-> %s...", prefixMsg, sbom.CiBuild, sbom.CiRun, sbom.Artifact, dst)
			bytes, err := api.SbomDownload(factory, targetName, buildRun, format)
			fmt.Println()
			subcommands.DieNotNil(err)
			subcommands.DieNotNil(os.WriteFile(dst, bytes, 0o744))
		}
	}
}
