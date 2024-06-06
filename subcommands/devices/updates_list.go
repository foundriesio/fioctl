package devices

import (
	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func doListUpdates(cmd *cobra.Command, args []string) {
	logrus.Debug("Showing device updates")
	t := tabby.New()
	t.AddHeader("ID", "TIME", "VERSION", "TARGET")
	d := getDeviceApi(cmd, args[0])
	var ul *client.UpdateList
	for {
		var err error
		if ul == nil {
			ul, err = d.ListUpdates()
		} else {
			if ul.Next != nil {
				ul, err = d.ListUpdatesCont(*ul.Next)
			} else {
				break
			}
		}
		subcommands.DieNotNil(err)
		for _, update := range ul.Updates {
			t.AddLine(update.CorrelationId, update.Time, update.Version, update.Target)
			listLimit -= 1
			if listLimit == 0 {
				break
			}
		}
		if listLimit == 0 {
			break
		}
	}
	t.Print()
}
