package keys

import (
	"fmt"
	"os"
	"time"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun       bool
	changeReason string
)

func init() {
	resign := &cobra.Command{
		Use:    "resign-root <offline key archive>",
		Short:  "Re-sign the Factory's TUF root metadata",
		Hidden: true,
		Run:    doResignRoot,
		Args:   cobra.ExactArgs(1),
		Deprecated: `it will be removed in the future.
Please, use a more secure way to keep your TUF root role fresh:
- rotate the root key using "fioctl keys tuf rotate-offline-key --role=root --keys=<offline-creds.tgz>".
`,
	}
	resign.Flags().BoolVarP(&dryRun, "dryrun", "", false, "Just print what the new root.json will look like and exit")
	resign.Flags().StringVarP(&changeReason, "changelog", "m", "", "Reason for resigning root.json. Saved in root metadata for tracking change history")
	cmd.AddCommand(resign)
}

func doResignRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	subcommands.AssertWritable(credsFile)
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)

	user, err := api.UserAccessDetails(factory, "self")
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Version += 1

	curPk, err := findTufRootSigner(root, creds)
	subcommands.DieNotNil(err)
	fmt.Println("= Current root:", curPk.Id)

	if len(changeReason) == 0 {
		changeReason = "resigning root.json"
	}
	root.Signed.Reason = &client.RootChangeReason{
		PolisId:   user.PolisId,
		Message:   changeReason,
		Timestamp: time.Now(),
	}

	fmt.Println("= Resigning root.json")
	removeUnusedTufKeys(root)
	subcommands.DieNotNil(signTufRoot(root, *curPk))

	bytes, err := subcommands.MarshalIndent(root, "", "  ")
	subcommands.DieNotNil(err)

	if dryRun {
		fmt.Println(string(bytes))
		return
	}

	fmt.Println("= Uploading new root.json")
	body, err := api.TufRootPost(factory, bytes)
	if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 409 {
		fmt.Println("ERROR: Your production root role is out of sync. Please run `fioctl keys rotate-root --sync-prod` to fix this.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println("\nERROR: ", err)
		fmt.Println(body)
		os.Exit(1)
	}
	// backfill this new key
	subcommands.DieNotNil(syncProdRoot(factory, root, creds, nil))
}
