package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"cloud.google.com/go/pubsub"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "listen <label> <creds file>",
		Short: "Listen to events sent to a pull queue",
		Args:  cobra.ExactArgs(2),
		Run:   doListen,
		Long: `Listens to pull queue events. This command is useful for debugging or as a 
reference implementation of queue listener.`,
	})
}

func subscriptionName(credsFile string) (string, string) {
	buf, err := os.ReadFile(credsFile)
	subcommands.DieNotNil(err)
	var config map[string]string
	err = json.Unmarshal(buf, &config)
	subcommands.DieNotNil(err)
	name, ok := config["fio-subscription-name"]
	if !ok {
		subcommands.DieNotNil(errors.New("Could not find \"fio-subscription-name\" attribute in credentials file"))
	}
	proj, ok := config["project_id"]
	if !ok {
		subcommands.DieNotNil(errors.New("Could not find \"project_id\" attribute in credentials file"))
	}
	return proj, name
}

func doListen(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listening to events for: %s", factory)

	proj, name := subscriptionName(args[1])

	ctx, cancel := context.WithCancel(cmd.Context())
	client, err := pubsub.NewClient(ctx, proj, option.WithCredentialsFile(args[1]))
	subcommands.DieNotNil(err)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Println("Exiting...")
			cancel()
		}
	}()

	fmt.Println("Listening for events...")
	sub := client.Subscription(name)
	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		fmt.Println(m.Attributes["event-type"], string(m.Data))
		m.Ack()
	})
	subcommands.DieNotNil(err)
}
