package teams

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
		Use:   "teams [<team_name>]",
		Short: "List teams belonging to a FoundriesFactory",
		Args:  cobra.RangeArgs(0, 1),
		Run:   doTeamsCommand,
	}
	subcommands.RequireFactory(cmd)
	return cmd
}

func doTeamsCommand(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		doList(subcommands.Login(cmd), viper.GetString("factory"))
	} else {
		doGetTeam(subcommands.Login(cmd), viper.GetString("factory"), args[0])
	}

}

func doList(api *client.Api, factory string) {
	logrus.Debugf("Listing FoundriesFactory teams for %s", factory)

	teams, err := api.TeamsList(factory)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("NAME", "DESCRIPTION")
	for _, team := range teams {
		t.AddLine(team.Name, team.Description)
	}
	t.Print()
}

func doGetTeam(api *client.Api, factory, team_name string) {
	team, err := api.TeamDetails(factory, team_name)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("NAME", "DESCRIPTION")
	t.AddLine(team.Name, team.Description)
	t.AddLine()

	t.AddHeader("SCOPES")
	for _, scope := range team.Scopes {
		t.AddLine(scope)
	}
	t.AddLine()

	t.AddHeader("DEVICE GROUPS")
	for _, group := range team.Groups {
		t.AddLine(group)
	}
	t.AddLine()

	t.AddHeader("ID", "NAME")
	for _, member := range team.Members {
		t.AddLine(member.PolisId, member.Name)
	}
	t.Print()
}
