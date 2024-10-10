package keys

import (
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
	show.Flags().BoolVarP(&showProd, "prod", "", false, "Show the production version")
	tufCmd.AddCommand(show)

	legacyShow := &cobra.Command{
		Use:   "show-root",
		Short: "Show the Factory's TUF root metadata",
		Deprecated: `it has moved to a new place.
Instead, use "fioctl keys tuf show-root".
`,
		Hidden: true,
		Run:    doShowRoot,
	}
	legacyShow.Flags().BoolVarP(&showProd, "prod", "", false, "Show the production version")
	cmd.AddCommand(legacyShow)
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
	bytes, err := subcommands.MarshalIndent(root, "", "  ")
	subcommands.DieNotNil(err)
	fmt.Println(string(bytes))
}
