package ci

import (
	"fmt"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:    "status",
		Short:  "Show a break down RUNNING and QUEUED CI runs",
		Hidden: true, // only usable by Factory admins that know about it
		Run:    doStatus,
	})
}

func doStatus(cmd *cobra.Command, args []string) {
	health, err := api.CiRunHealth()
	subcommands.DieNotNil(err)
	fmt.Println("Queued Runs")
	t := subcommands.Tabby(1, "CREATED", "PROJECT", "BUILD", "RUN", "HOST TAG")
	for _, r := range health.Queued {
		t.AddLine(r.Created, r.Project, r.Build, r.Run, r.HostTag)
	}
	t.Print()

	fmt.Println("Active Runs")
	t = subcommands.Tabby(1, "WORKER", "CREATED", "PROJECT", "BUILD", "RUN", "HOST TAG")
	for worker, runs := range health.Running {
		for _, r := range runs {
			t.AddLine(worker, r.Created, r.Project, r.Build, r.Run, r.HostTag)
		}
	}
	t.Print()
}
