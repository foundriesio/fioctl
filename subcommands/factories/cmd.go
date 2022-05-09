package factories

import (
	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
)

var (
	api   *client.Api
	admin bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "factories",
		Short:  "List factories a user is a member of.",
		Hidden: true, // Only useful support work
		Run:    doFactories,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = subcommands.Login(cmd)
		},
	}
	cmd.Flags().BoolVarP(&admin, "admin", "", false, "Show all factories")
	return cmd
}

func doFactories(cmd *cobra.Command, args []string) {
	factories, err := api.FactoriesList(admin)
	subcommands.DieNotNil(err)
	t := tabby.New()
	t.AddHeader("NAME", "ID")
	for _, f := range factories {
		t.AddLine(f.Name, f.Id)
	}
	t.Print()
}
