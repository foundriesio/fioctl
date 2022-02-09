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
		Use:   "users [<user_id>]",
		Short: "List users with access to a FoundriesFactory",
		Args:  cobra.RangeArgs(0, 1),
		Run:   doUserCommand,
	}
	subcommands.RequireFactory(cmd)
	return cmd
}

func doUserCommand(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		doList(subcommands.Login(cmd), viper.GetString("factory"))
	} else {
		doGetUser(subcommands.Login(cmd), viper.GetString("factory"), args[0])
	}

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

func doGetUser(api *client.Api, factory, user_id string) {
	user, err := api.UserAccessDetails(factory, user_id)
	subcommands.DieNotNil(err)
	t := tabby.New()
	t.AddHeader("ID", "NAME", "ROLE")
	t.AddLine(user.PolisId, user.Name, user.Role)

	t.AddLine()
	t.AddHeader("TEAMS")
	for _, team := range user.Teams {
		t.AddLine(team)
	}
	t.AddLine()
	t.AddHeader("EFFECTIVE SCOPES")
	for _, scope := range user.EffectiveScopes {
		t.AddLine(scope)
	}
	t.Print()
}
