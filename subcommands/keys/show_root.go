package keys

import (
	"encoding/json"
	"fmt"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	showProd bool
)

func init() {
	show := &cobra.Command{
		Use:   "show-root",
		Short: "Show the Factory's TUF root metadata",
		Run:   doShowRoot,
	}
	subcommands.RequireFactory(show)
	show.Flags().BoolVarP(&showProd, "prod", "", false, "Show the production version")
	cmd.AddCommand(show)
}

func doShowRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	var err error
	var root *client.AtsTufRoot
	if showProd {
		root, err = api.TufProdRootGet(factory)
	} else {
		root, err = api.TufRootGet(factory)
	}
	subcommands.DieNotNil(err)
	bytes, err := json.MarshalIndent(root, "", "  ")
	subcommands.DieNotNil(err)
	fmt.Println(string(bytes))
}
