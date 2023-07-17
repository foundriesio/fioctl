package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/karrick/godiff"
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
	review.Flags().BoolP("raw", "", false, "Show the raw root.json")
	review.Flags().BoolP("diff", "", false, "Show the unified diff between current and staged root.json")
	review.MarkFlagsMutuallyExclusive("raw", "diff")
	review.Flags().BoolP("prod", "", false, "Show the production root.json")
	tufUpdatesCmd.AddCommand(review)
}

func doTufUpdatesReview(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	showRaw, _ := cmd.Flags().GetBool("raw")
	showDiff, _ := cmd.Flags().GetBool("diff")
	showProd, _ := cmd.Flags().GetBool("prod")
	if showProd && !showRaw && !showDiff {
		subcommands.DieNotNil(errors.New(
			"If the flag 'prod' is set then one of the flags [raw diff] must also be set",
		))
	}

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	oldCiRoot, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, false)

	if showRaw || showDiff {
		if updates.Status == client.TufRootUpdatesStatusNone {
			subcommands.DieNotNil(errors.New("There are no TUF root updates in progress."))
		}
		var rootToShow *client.AtsTufRoot
		if newProdRoot == nil {
			// No effective changes yet, but we know how the prod root would look like
			newProdRoot = genProdTufRoot(newCiRoot)
		}

		if showProd {
			rootToShow = newProdRoot
		} else {
			rootToShow = newCiRoot
		}

		if showRaw {
			bytes, err := subcommands.MarshalIndent(rootToShow, "", "  ")
			subcommands.DieNotNil(err)
			fmt.Println(string(bytes))
		} else {
			var baseRootToShow *client.AtsTufRoot
			if showProd {
				if updates.Current.ProdRoot != "" {
					subcommands.DieNotNil(
						json.Unmarshal([]byte(updates.Current.ProdRoot), &baseRootToShow),
						"Current prod root",
					)
				} else {
					// First rotation, old prod root equals old CI root
					baseRootToShow = oldCiRoot
				}
			} else {
				baseRootToShow = oldCiRoot
			}

			before, err := subcommands.MarshalIndent(baseRootToShow, "", "  ")
			subcommands.DieNotNil(err)
			after, err := subcommands.MarshalIndent(rootToShow, "", "  ")
			subcommands.DieNotNil(err)
			diff := godiff.Strings(
				strings.Split(string(before), "\n"),
				strings.Split(string(after), "\n"),
			)
			for _, line := range diff {
				if line[0] == '+' {
					color.Green(line)
				} else if line[0] == '-' {
					color.Red(line)
				} else {
					fmt.Println(line)
				}
			}
		}
	} else if updates.Status == client.TufRootUpdatesStatusNone {
		fmt.Println("There are no TUF root updates in progress.")
		// There can be no errors for existing root: it is impossible to upload erroneous TUF root.
		if len(updates.Issues.Warnings) > 0 {
			fmt.Println("\nThese updates to your existing TUF metadata are recommended:")
			for _, issue := range updates.Issues.Warnings {
				fmt.Printf(" - %s\n", issue.Message)
			}
		}
	} else {
		fmt.Println("The following TUF root updates are staged for your factory:")
		for _, amendment := range updates.Amendments {
			fmt.Printf(" - %s\n", amendment.Message)
		}
		if len(updates.Issues.Errors) > 0 {
			fmt.Println("\nThese updates to your staged TUF root are mandatory before applying it:")
			for _, issue := range updates.Issues.Errors {
				fmt.Printf(" - %s\n", issue.Message)
			}
		}
		if len(updates.Issues.Warnings) > 0 {
			fmt.Println("\nThese updates to your staged TUF root are recommended:")
			for _, issue := range updates.Issues.Warnings {
				fmt.Printf(" - %s\n", issue.Message)
			}
		}

		if updates.Status == client.TufRootUpdatesStatusApplying {
			fmt.Println(`
These changes are currently being applied. No more changes can be staged.
If the previous 'fioctl keys tuf updates apply' command failed, please, try to run it again.`)
		} else {
			fmt.Println(`
Once your are satisfied with your TUF updates, please, run 'fioctl keys tuf updates apply'.
If you want to cancel staged TUF updates, please, run 'fioctl keys tuf updates cancel'.`)
		}
	}
}
