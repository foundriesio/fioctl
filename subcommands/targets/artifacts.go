package targets

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "artifacts <target> [<artifact name>]",
		Short: "Show artifacts created in CI for a Target",
		Run:   doArtifacts,
		Args:  cobra.RangeArgs(1, 2),
		Example: `
  # List all artifacts for Target 12
  fioctl targets artifacts 12

  # Dump console.log artifact to STDOUT
  fioctl targets artifacts 12 publish-compose-apps/console.log
  
  # Download an artifact. Progress is printed to STDERR, the contents is 
  # re-directed /tmp/tmp.gz
  fioctl-linux-amd64 targets artifacts 207 \
    raspberrypi3-64/lmp-factory-image-raspberrypi3-64.wic.gz >/tmp/tmp.gz
`,
	})
}

func listArtifacts(factory string, target int) {
	runs, err := api.JobservRuns(factory, target)
	subcommands.DieNotNil(err)
	for _, run := range runs {
		fmt.Println(run.Name, run.Url)
		run, err := api.JobservRun(run.Url)
		subcommands.DieNotNil(err)
		stripLen := len(run.Url) - len(run.Name) - 1
		for _, a := range run.Artifacts {
			fmt.Println(a[stripLen:])
		}
	}
}

type DlStatus struct {
	Total   int64
	Current int64
	Width   int64
	LastMsg time.Time
}

func (d *DlStatus) Write(p []byte) (int, error) {
	n := len(p)
	d.Current += int64(n)

	now := time.Now()
	elapsed := now.Sub(d.LastMsg)
	if elapsed.Seconds() < 1 && d.Current != d.Total {
		return n, nil
	}
	d.LastMsg = now
	current := int64((float64(d.Current) / float64(d.Total)) * float64(d.Width))
	fmt.Fprintf(
		os.Stderr,
		"[%s%s] %d of %d\r",
		strings.Repeat("=", int(current)),
		strings.Repeat(" ", int(d.Width-current)),
		d.Current,
		d.Total)
	return n, nil
}

func downloadArtifact(factory string, target int, artifact string) {
	firstSlash := strings.Index(artifact, "/")
	if firstSlash < 1 {
		fmt.Println("ERROR: Invalid artifact path:", artifact)
		os.Exit(1)
	}
	run := artifact[0:firstSlash]
	artifact = artifact[firstSlash+1:]

	resp, err := api.JobservRunArtifact(factory, target, run, artifact)
	subcommands.DieNotNil(err)

	status := DlStatus{resp.ContentLength, 0, 20, time.Now()}
	written, err := io.Copy(os.Stdout, io.TeeReader(resp.Body, &status))
	fmt.Fprintln(os.Stderr)
	subcommands.DieNotNil(err)
	if written != resp.ContentLength {
		fmt.Fprintf(os.Stderr, "ERROR: read %d bytes, expected %d bytes\n", written, resp.ContentLength)
		os.Exit(1)
	}
}

func doArtifacts(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	if len(args) == 0 {
		logrus.Debugf("Showing all testing done for factory: %s", factory)
		listAll(factory)
		os.Exit(0)
	}

	target, err := strconv.Atoi(args[0])
	subcommands.DieNotNil(err)
	if len(args) == 1 {
		logrus.Debugf("Showing target artifacts for %s %d", factory, target)
		listArtifacts(factory, target)
	} else {
		artifact := args[1]
		logrus.Debugf("Downloading artifact %s %d %s", factory, target, artifact)
		downloadArtifact(factory, target, artifact)
	}
}
