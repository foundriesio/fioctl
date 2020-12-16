package subcommands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	toml "github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"

	"github.com/foundriesio/fioctl/client"
)

// Aktualizr puts all config files into a single lexographically sorted map.
// We have to make sure this file is parsed *after* sota.toml.
const (
	FIO_TOML_NAME        = "z-50-fioctl.toml"
	FIO_COMPOSE_APPS_DIR = "/var/sota/compose-apps"
	FIO_TOML_ONCHANGED   = "/usr/share/fioconfig/handlers/aktualizr-toml-update"
)

type SetConfigOptions struct {
	Reason      string
	FileArgs    []string
	IsRawFile   bool
	SetFunc     func(client.ConfigCreateRequest) error
	EncryptFunc func(string) string
}

func SetConfig(opts *SetConfigOptions) {
	cfg := client.ConfigCreateRequest{Reason: opts.Reason}
	if opts.IsRawFile {
		if len(opts.FileArgs) != 1 {
			DieNotNil(fmt.Errorf("Raw file only accepts one file argument"))
		}
		ReadConfig(opts.FileArgs[0], &cfg)
	} else {
		for _, keyval := range opts.FileArgs {
			parts := strings.SplitN(keyval, "=", 2)
			if len(parts) != 2 {
				DieNotNil(fmt.Errorf("Invalid file=content argument: %s", keyval))
			}
			cfg.Files = append(cfg.Files, client.ConfigFile{Name: parts[0], Value: parts[1]})
		}
	}

	if opts.EncryptFunc != nil {
		for i := range cfg.Files {
			file := &cfg.Files[i]
			if !file.Unencrypted {
				file.Value = opts.EncryptFunc(file.Value)
			}
		}
	}

	DieNotNil(opts.SetFunc(cfg))
}

type LogConfigsOptions struct {
	Limit         int
	ShowAppliedAt bool
	ListFunc      func() (*client.DeviceConfigList, error)
	ListContFunc  func(string) (*client.DeviceConfigList, error)
}

func LogConfigs(opts *LogConfigsOptions) {
	var dcl *client.DeviceConfigList
	listLimit := opts.Limit
	for {
		var err error
		if dcl == nil {
			dcl, err = opts.ListFunc()
		} else {
			if dcl.Next != nil {
				dcl, err = opts.ListContFunc(*dcl.Next)
			} else {
				return
			}
		}
		DieNotNil(err)
		for _, cfg := range dcl.Configs {
			PrintConfig(&cfg, opts.ShowAppliedAt, true, "")
			if listLimit -= 1; listLimit == 0 {
				return
			} else {
				fmt.Println("")
			}
		}
	}
}

func ReadConfig(configFile string, cfg *client.ConfigCreateRequest) {
	var content []byte
	var err error

	if configFile == "-" {
		logrus.Debug("Reading config from STDIN")
		content, err = ioutil.ReadAll(os.Stdin)
	} else {
		content, err = ioutil.ReadFile(configFile)
	}

	DieNotNil(err, "Unable to read config file:")
	DieNotNil(json.Unmarshal(content, cfg), "Unable to parse config file:")
}

func PrintConfig(cfg *client.DeviceConfig, showAppliedAt, highlightFirstLine bool, indent string) {
	printf := func(format string, a ...interface{}) {
		fmt.Printf(indent+format, a...)
	}

	if highlightFirstLine {
		firstLine := color.New(color.FgYellow)
		firstLine.Printf(indent+"Created At:    %s\n", cfg.CreatedAt)
	} else {
		printf("Created At:    %s\n", cfg.CreatedAt)
	}
	if showAppliedAt {
		printf("Applied At:    %s\n", cfg.AppliedAt)
	}
	printf("Change Reason: %s\n", cfg.Reason)
	printf("Files:\n")
	for _, f := range cfg.Files {
		if len(f.OnChanged) == 0 {
			printf("\t%s\n", f.Name)
		} else {
			printf("\t%s - %v\n", f.Name, f.OnChanged)
		}
		if f.Unencrypted {
			for _, line := range strings.Split(f.Value, "\n") {
				printf("\t | %s\n", line)
			}
		}
	}
}

type SetUpdatesConfigOptions struct {
	UpdateTags     string
	UpdateApps     string
	SetComposeApps bool
	ComposeAppsDir string
	IsDryRun       bool
	IsForced       bool
	Device         *client.Device
	ListFunc       func() (*client.DeviceConfigList, error)
	SetFunc        func(client.ConfigCreateRequest, bool) error
}

func SetUpdatesConfig(opts *SetUpdatesConfigOptions) {
	DieNotNil(validateUpdateArgs(opts))

	dcl, err := opts.ListFunc()
	if !opts.IsForced {
		DieNotNil(err, "Failed to fetch existing config changelog (override with --force):")
	}
	sota, err := loadSotaConfig(dcl)
	if !opts.IsForced {
		DieNotNil(err, "Invalid FIO toml file (override with --force):")
	}

	if opts.UpdateApps == "" && opts.UpdateTags == "" && !opts.SetComposeApps {
		if opts.Device != nil {
			fmt.Println("= Reporting to server with")
			fmt.Println(" Tags: ", strings.Join(opts.Device.Tags, ","))
			fmt.Println(" Apps: ", strings.Join(opts.Device.DockerApps, ","))
			fmt.Println("")
		}
		fmt.Println("= Configured overrides")
		fmt.Println(sota)
		return
	}

	configuredApps := sota.GetDefault("pacman.docker_apps", "").(string)
	configuredTags := sota.GetDefault("pacman.tags", "").(string)
	configuredMgr := sota.GetDefault("pacman.packagemanager", "").(string)

	changed := false
	if opts.UpdateApps != "" && configuredApps != opts.UpdateApps {
		if strings.TrimSpace(opts.UpdateApps) == "," {
			opts.UpdateApps = ""
		}
		fmt.Printf("Changing apps from: [%s] -> [%s]\n", configuredApps, opts.UpdateApps)
		sota.Set("pacman.docker_apps", opts.UpdateApps)
		sota.Set("pacman.compose_apps", opts.UpdateApps)
		changed = true
	}
	if opts.UpdateTags != "" && configuredTags != opts.UpdateTags {
		if strings.TrimSpace(opts.UpdateTags) == "," {
			opts.UpdateTags = ""
		}
		fmt.Printf("Changing tags from: [%s] -> [%s]\n", configuredTags, opts.UpdateTags)
		sota.Set("pacman.tags", opts.UpdateTags)
		changed = true
	}
	if opts.SetComposeApps && configuredMgr != "ostree+compose_apps" {
		fmt.Printf("Changing packagemanager to %s\n", "ostree+compose_apps")
		sota.Set("pacman.type", "ostree+compose_apps")
		if opts.ComposeAppsDir == "" {
			opts.ComposeAppsDir = FIO_COMPOSE_APPS_DIR
		}
		sota.Set("pacman.compose_apps_root", opts.ComposeAppsDir)
		// the device might be running DockerApps that were set in /var/sota/sota.toml
		// by lmp-device-register, so fallback to what its reporting if we don't find
		// override values set:
		defaultApps := ""
		if opts.Device != nil {
			defaultApps = strings.Join(opts.Device.DockerApps, ",")
		}
		sota.Set("pacman.compose_apps", sota.GetDefault("pacman.docker_apps", defaultApps))
		changed = true
	} else if opts.ComposeAppsDir != "" {
		fmt.Println("Can only change compose apps dir when migrating to compose apps.")
	}

	if !changed {
		DieNotNil(fmt.Errorf(
			"No changes found. Device is already configured with the specified options."))
	}

	newToml, err := sota.ToTomlString()
	DieNotNil(err, "Unable to encode toml:")

	cfg := client.ConfigCreateRequest{
		Reason: "Override aktualizr-lite update configuration ",
		Files: []client.ConfigFile{
			{
				Name:        FIO_TOML_NAME,
				Unencrypted: true,
				OnChanged:   []string{"/usr/share/fioconfig/handlers/aktualizr-toml-update"},
				Value:       newToml,
			},
		},
	}
	if opts.IsDryRun {
		fmt.Println(newToml)
	} else {
		DieNotNil(opts.SetFunc(cfg, opts.IsForced))
	}
}

func loadSotaConfig(dcl *client.DeviceConfigList) (sota *toml.Tree, err error) {
	found := false
	if dcl != nil && len(dcl.Configs) > 0 {
		for _, cfgFile := range dcl.Configs[0].Files {
			if cfgFile.Name == FIO_TOML_NAME {
				sota, err = toml.Load(cfgFile.Value)
				if err != nil {
					err = fmt.Errorf("Unable to decode toml: %w\n- TOML is: %s", err, cfgFile.Value)
				}
				found = true
				break
			}
		}
	}

	if !found {
		logrus.Debugf("Not found a FIO toml in the latest config")
	}
	// In case if FIO TOML file is missing or an error - return an empty one.
	// Let a caller decide what to do in case of an error.
	if !found || err != nil {
		sota, _ = toml.Load("[pacman]")
	}
	return
}

func validateUpdateArgs(opts *SetUpdatesConfigOptions) error {
	// Validate the inputs: Must be alphanumeric, a dash, underscore, or comma
	pattern := `^[a-zA-Z0-9-_,]+$`
	re := regexp.MustCompile(pattern)
	if len(opts.UpdateApps) > 0 && !re.MatchString(opts.UpdateApps) {
		return fmt.Errorf("Invalid value for apps: %s\nMust be %s", opts.UpdateApps, pattern)
	}
	if len(opts.UpdateTags) > 0 && !re.MatchString(opts.UpdateTags) {
		return fmt.Errorf("Invalid value for tags: %s\nMust be %s", opts.UpdateTags, pattern)
	}
	return nil
}
