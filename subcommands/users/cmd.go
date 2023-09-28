package users

import (
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		t.AddLine(team.Name)
		var scopes []string
		for _, s := range team.Scopes {
			scopes = append(scopes, s[strings.Index(s, ":")+1:])
		}
		if len(scopes) > 0 {
			t.AddLine("\tScopes: " + strings.Join(scopes, ", "))
		}
		if len(team.Groups) > 0 {
			t.AddLine("\tGroups: " + strings.Join(team.Groups, ", "))
		}
		t.AddLine()
	}
	t.AddLine()
	t.AddHeader("EFFECTIVE SCOPES")
	for _, scope := range user.EffectiveScopes {
		t.AddLine(scope[strings.Index(scope, ":")+1:])
	}
	t.Print()
}
