package keys

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dryRun bool

func init() {
	resign := &cobra.Command{
		Use:   "resign-root <offline key archive>",
		Short: "Re-sign the Factory's TUF root metadata",
		Run:   doResignRoot,
		Args:  cobra.ExactArgs(1),
		Long:  "The root metadata, root.json expires. Re-signing bumps the expiriation date.",
	}
	subcommands.RequireFactory(resign)
	resign.Flags().BoolVarP(&dryRun, "dryrun", "", false, "Just print what the new root.json will look like and exit")
	cmd.AddCommand(resign)
}

func doResignRoot(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	assertWritable(credsFile)
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	root.Signed.Expires = time.Now().AddDate(1, 0, 0).UTC().Round(time.Second) // 1 year validity
	root.Signed.Version += 1

	curid, curPk, err := findRoot(*root, creds)
	fmt.Println("= Current root:", curid)
	subcommands.DieNotNil(err)

	fmt.Println("= Resigning root.json")
	signers := []TufSigner{
		{Id: curid, Key: curPk},
	}
	removeUnusedKeys(root)
	subcommands.DieNotNil(signRoot(root, signers...))

	bytes, err := json.MarshalIndent(root, "", "  ")
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
	subcommands.DieNotNil(syncProdRoot(factory, *root, creds))
}
