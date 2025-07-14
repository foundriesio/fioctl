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
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "tests [<target> [<test-id> [<artifact name>]]]",
		Short: "Show testing done against a Target",
		Run:   doShowTests,
		Args:  cobra.RangeArgs(0, 3),
		Example: `
  # List all testing performed in the Factory
  fioctl targets tests

  # Show tests run against Target 12
  fioctl targets tests 12

  # Show details of a specific test
  fioctl targets tests 12 <test-id>

  # Display a test artifact
  fioctl targets tests 12 <test-id> console.log
`,
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

func listAll(factory string) {
	versions, err := api.TargetTestingApi(factory).Versions()
	subcommands.DieNotNil(err)
	fmt.Println("Tested Targets:")
	for _, ver := range versions {
		fmt.Println(" ", ver)
	}
}

func list(factory string, target int) {
	tapi := api.TargetTestingApi(factory)
	t := tabby.New()
	t.AddHeader("NAME", "STATUS", "ID", "CREATED AT", "DEVICE")

	var tl *client.TargetTestList
	for {
		var err error
		if tl == nil {
			tl, err = tapi.Tests(target)
		} else {
			if tl.Next != nil {
				tl, err = api.TargetTestsCont(*tl.Next)
			} else {
				break
			}
		}
		subcommands.DieNotNil(err)
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
	test, err := api.TargetTestingApi(factory).TestResults(target, testId)
	subcommands.DieNotNil(err)
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

	if len(test.Artifacts) > 0 {
		fmt.Println("Artifacts:")
		for _, a := range test.Artifacts {
			fmt.Println("\t", a)
		}
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

	if len(args) == 0 {
		logrus.Debugf("Showing all testing done for Factory: %s", factory)
		listAll(factory)
		os.Exit(0)
	}

	target, err := strconv.Atoi(args[0])
	subcommands.DieNotNil(err)
	if len(args) == 1 {
		logrus.Debugf("Showing Target testing for %s %d", factory, target)
		list(factory, target)
	} else if len(args) == 2 {
		testId := args[1]
		logrus.Debugf("Showing Target test results for %s %d - %s", factory, target, testId)
		show(factory, target, testId)
	} else {
		testId := args[1]
		artifact := args[2]
		logrus.Debugf("Showing Target test artifacts for %s %d - %s / %s", factory, target, testId, artifact)
		content, err := api.TargetTestingApi(factory).TestArtifact(target, testId, artifact)
		subcommands.DieNotNil(err)
		os.Stdout.Write(*content)
	}
}
