package subcommands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"

	"github.com/foundriesio/fioctl/client"
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
