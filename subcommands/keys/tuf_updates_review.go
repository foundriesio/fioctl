package keys

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	review := &cobra.Command{
		Use:   "review",
		Short: "Show the Factory's TUF root metadata",
		Run:   doTufUpdatesReview,
	}
	review.Flags().BoolP("prod", "", false, "Show the production version")
	tufUpdatesCmd.AddCommand(review)
}

func doTufUpdatesReview(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	showProd, _ := cmd.Flags().GetBool("prod")

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	var rootToShow, newCiRoot, newProdRoot *client.AtsTufRoot
	_, newCiRoot = checkTufRootUpdatesStatus(updates, false)
	if updates.Updated.ProdRoot != "" {
		subcommands.DieNotNil(
			json.Unmarshal([]byte(updates.Updated.ProdRoot), &newProdRoot), "Updated prod root",
		)
	}
	if newProdRoot == nil {
		fmt.Println("There are no staged TUF updates.")
		return
	}

	if showProd {
		rootToShow = newProdRoot
	} else {
		rootToShow = newCiRoot
	}
	bytes, err := subcommands.MarshalIndent(rootToShow, "", "  ")
	subcommands.DieNotNil(err)
	fmt.Println(string(bytes))
}
