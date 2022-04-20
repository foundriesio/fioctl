package events

import (
	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured event queues",
		Run:     doList,
	})
}

func doList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing event queues for: %s", factory)

	queues, err := api.EventQueuesList(factory)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("LABEL", "TYPE", "PUSH URL")
	for _, queue := range queues {
		t.AddLine(queue.Label, queue.Type, queue.PushUrl)
	}
	t.Print()
}
