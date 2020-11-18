package users

import (
	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "List users with access to a factory",
		Run: func(cmd *cobra.Command, args []string) {
			doList(subcommands.Login(cmd), viper.GetString("factory"))
		},
	}
	subcommands.RequireFactory(cmd)
	return cmd
}

func doList(api *client.Api, factory string) {
	logrus.Debugf("Listing factory users for %s", factory)

	users, err := api.UsersList(factory)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("ID", "NAME", "ROLE")
	for _, user := range users {
		t.AddLine(user.PolisId, user.Name, user.Role)
	}
	t.Print()
}
