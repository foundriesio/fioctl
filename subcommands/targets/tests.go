package targets

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "tests <target> [<test-id>]",
		Short: "Show testing done against a target",
		Run:   doShowTests,
		Args:  cobra.RangeArgs(1, 2),
	})
}

func timestamp(ts float32) string {
	if ts == 0 {
		return ""
	}
	secs := int64(ts)
	nsecs := int64((ts - float32(secs)) * 1e9)
	return time.Unix(secs, nsecs).UTC().String()
}

func list(factory string, target int) {
	t := tabby.New()
	t.AddHeader("NAME", "STATUS", "ID", "CREATED AT", "DEVICE")

	var tl *client.TargetTestList
	for {
		var err error
		if tl == nil {
			tl, err = api.TargetTests(factory, target)
		} else {
			if tl.Next != nil {
				tl, err = api.TargetTestsCont(*tl.Next)
			} else {
				break
			}
		}
		if err != nil {
			fmt.Print("ERROR: ")
			fmt.Println(err)
			os.Exit(1)
		}
		for _, test := range tl.Tests {
			created := timestamp(test.CreatedOn)
			name := test.DeviceUUID
			if len(test.DeviceName) > 0 {
				name = test.DeviceName
			}
			t.AddLine(test.Name, test.Status, test.Id, created, name)
		}
	}
	t.Print()
}

func show(factory string, target int, testId string) {
	test, err := api.TargetTestResults(factory, target, testId)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Name:     ", test.Name)
	fmt.Println("Status:   ", test.Status)
	fmt.Println("Created:  ", timestamp(test.CreatedOn))
	fmt.Println("Completed:", timestamp(test.CompletedOn))
	name := test.DeviceUUID
	if len(test.DeviceName) > 0 {
		name = test.DeviceName
	}
	fmt.Println("Device:   ", name)
	if len(test.Details) > 0 {
		fmt.Println("Details:")
		fmt.Println(test.Details)
	}
	if test.Results != nil {
		fmt.Println("")
		t := tabby.New()
		t.AddHeader("TEST RESULT", "STATUS")
		for _, result := range test.Results {
			t.AddLine(result.Name, result.Status)
		}
		t.Print()
	}
}

func doShowTests(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	target, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Print("ERROR:")
		fmt.Println(err)
		os.Exit(1)
	}
	if len(args) == 1 {
		logrus.Debugf("Showing target testing for %s %d", factory, target)
		list(factory, target)
	} else {
		testId := args[1]
		logrus.Debugf("Showing target test results for %s %d - %s", factory, target, testId)
		show(factory, target, testId)
	}
}
