package waves

import (
	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show available waves",
		Run:   doListWaves,
	}
	cmd.AddCommand(listCmd)
	listCmd.Flags().Uint64P("limit", "n", 20, "Limit the number of results displayed.")
	listCmd.Flags().IntP("page", "p", 1, "Page of waves to display when pagination is needed")
}

func doListWaves(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	limit, _ := cmd.Flags().GetUint64("limit")
	showPage, _ := cmd.Flags().GetInt("page")
	logrus.Debugf("Showing a list of waves for %s", factory)

	lst, err := api.FactoryListWaves(factory, limit, showPage)
	subcommands.DieNotNil(err)

	t := tabby.New()
	t.AddHeader("NAME", "VERSION", "TAG", "STATUS", "CREATED AT", "FINISHED AT")
	for _, wave := range lst.Waves {
		t.AddLine(
			wave.Name,
			wave.Version,
			wave.Tag,
			wave.Status,
			wave.ChangeMeta.CreatedAt,
			wave.ChangeMeta.UpdatedAt,
		)
	}
	t.Print()
	subcommands.ShowPages(showPage, lst.Next)
}
