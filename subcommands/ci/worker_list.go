package ci

import (
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:    "workers",
		Short:  "List all CI workers know to the Factory",
		Hidden: true, // only usable by Factory admins that know about it
		Run:    doWorkerList,
	})
}

func doWorkerList(cmd *cobra.Command, args []string) {
	workers, err := api.CiWorkersList()
	subcommands.DieNotNil(err)
	giB := 1024 * 1024 * 1024
	t := tabby.New()
	t.AddHeader("NAME", "ENLISTED", "ONLINE", "TAGS", "CONCURRENT RUNS", "CPUS", "MEMORY", "DISTRO")
	for _, w := range workers {
		tags := strings.Join(w.HostTags, ",")
		cpus := fmt.Sprintf("%d %s", w.CpuTotal, w.CpuType)
		mem := fmt.Sprintf("%dGib", w.MemTotal/giB)

		t.AddLine(w.Name, w.Enlisted, w.Online, tags, w.ConcurrentRuns, cpus, mem, w.Distro)
	}
	t.Print()
}
