package events

import (
	"io/ioutil"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "mk-push <label> <url>",
		Short: "Create an event queue that will ingest events at the URL",
		Args:  cobra.ExactArgs(2),
		Run:   doCreatePush,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "mk-pull <label> <pubsub creds file>",
		Short: "Create a message queue that can be polled for events",
		Args:  cobra.ExactArgs(2),
		Run:   doCreatePull,
		Long: `Create a message queue that can be polled for events via the Google PubSub API:

  https://cloud.google.com/pubsub/docs/reference/libraries 

The command creates a credentials file to a scoped service account capable of 
polling the resulting PubSub subscription.`,
	})
}

func doCreatePush(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Create a push queue for: %s", factory)

	queue := client.EventQueue{
		Label:   args[0],
		Type:    "push",
		PushUrl: args[1],
	}

	_, err := api.EventQueuesCreate(factory, queue)
	subcommands.DieNotNil(err)
}

func doCreatePull(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Create a pull queue for: %s", factory)

	queue := client.EventQueue{
		Label: args[0],
		Type:  "pull",
	}

	creds, err := api.EventQueuesCreate(factory, queue)
	subcommands.DieNotNil(err)
	err = ioutil.WriteFile(args[1], creds, 0o700)
	subcommands.DieNotNil(err)
}
