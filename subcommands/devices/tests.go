package devices

import (
	"fmt"
	"os"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	logsCmd := &cobra.Command{
		Use:   "tests <device> [<test id> [<artifact name>]]",
		Short: "List tests results uploaded by a device",
		Run:   doTests,
		Args:  cobra.RangeArgs(1, 3),
	}
	cmd.AddCommand(logsCmd)
}

func timestamp(ts float32) string {
	if ts == 0 {
		return ""
	}
	secs := int64(ts)
	nsecs := int64((ts - float32(secs)) * 1e9)
	return time.Unix(secs, nsecs).UTC().String()
}

func doTests(cmd *cobra.Command, args []string) {
	switch len(args) {
	case 1:
		doTestList(cmd, args)
	case 2:
		doTestShow(cmd, args)
	case 3:
		doTestArtifact(cmd, args)
	default:
		panic("invalid number of args") // "impossible" because of cobra.RangeArgs
	}
}

func doTestList(cmd *cobra.Command, args []string) {
	name := args[0]
	d := getDevice(cmd, name)

	t := tabby.New()
	t.AddHeader("NAME", "STATUS", "ID", "CREATED AT")

	var tl *client.TargetTestList
	for {
		var err error
		if tl == nil {
			tl, err = d.Api.Tests()
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
			t.AddLine(test.Name, test.Status, test.Id, created)
		}
	}
	t.Print()
}

func doTestShow(cmd *cobra.Command, args []string) {
	name := args[0]
	testId := args[1]
	d := getDevice(cmd, name)

	result, err := d.Api.TestGet(testId)
	subcommands.DieNotNil(err)

	fmt.Println("Name:     ", result.Name)
	fmt.Println("Status:   ", result.Status)
	fmt.Println("Created:  ", timestamp(result.CreatedOn))
	fmt.Println("Completed:", timestamp(result.CompletedOn))

	if len(result.Details) > 0 {
		fmt.Println("Details:")
		fmt.Println(result.Details)
	}

	if len(result.Artifacts) > 0 {
		fmt.Println("Artifacts:")
		for _, a := range result.Artifacts {
			fmt.Println("\t", a)
		}
	}
	if len(result.Results) > 0 {
		fmt.Println("")
		t := tabby.New()
		t.AddHeader("TEST RESULT", "STATUS")
		for _, result := range result.Results {
			t.AddLine(result.Name, result.Status)
		}
		t.Print()
	}
}

func doTestArtifact(cmd *cobra.Command, args []string) {
	name := args[0]
	testId := args[1]
	artifact := args[2]
	d := getDevice(cmd, name)

	content, err := d.Api.TestResultArtifact(testId, artifact)
	subcommands.DieNotNil(err)
	os.Stdout.Write(*content)
}
