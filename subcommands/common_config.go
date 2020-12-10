package subcommands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/foundriesio/fioctl/client"
)

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

func PrintConfig(cfg *client.DeviceConfig, showAppliedAt, highlightFirstLine bool, indent string) {
	firstLine := fmt.Printf
	if highlightFirstLine {
		firstLine = color.New(color.FgYellow).Printf
	}
	printf := func(format string, a ...interface{}) {
		fmt.Printf(indent+format, a...)
	}

	firstLine(indent+"Created At:    %s\n", cfg.CreatedAt)
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
