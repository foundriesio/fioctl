package keys

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	copy := &cobra.Command{
		Use:    "copy-targets <offline key archive> <new targets archive>",
		Short:  "Copy the target signing credentials from the offline key archive",
		Hidden: true,
		Run:    doCopyTargets,
		Args:   cobra.ExactArgs(2),
		Long: `This command extracts the target signing credentials required for initializing 
waves into a new tarball so that the offline key archive isn't required for
rolling out production updates. This should be run after each target key
rotation and distributed to the operator in charge of production OTAs.`,
		Deprecated: `it will be removed in the future.
Please, use a new approach to rotate the targets key into a separate file:
	fioctl keys tuf rotate-offline-key --role=targets \
		--keys=<offline-creds.tgz> --targets-keys=<offline-targets-creds.tgz>
`,
	}
	cmd.AddCommand(copy)
}

func doCopyTargets(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	credsFile := args[0]
	creds, err := GetOfflineCreds(credsFile)
	subcommands.DieNotNil(err)

	root, err := api.TufRootGet(factory)
	subcommands.DieNotNil(err)

	targetsCreds, err := createTargetsCreds(factory, *root, creds)
	subcommands.DieNotNil(err)
	SaveCreds(args[1], targetsCreds)
}

func createTargetsCreds(factory string, root client.AtsTufRoot, creds OfflineCreds) (OfflineCreds, error) {
	targets := make(OfflineCreds)
	onlinePub, err := api.GetFoundriesTargetsKey(factory)
	subcommands.DieNotNil(err)
	for _, keyid := range root.Signed.Roles["targets"].KeyIDs {
		pubkey := root.Signed.Keys[keyid].KeyValue.Public
		if pubkey != onlinePub.KeyValue.Public {
			pubkey = strings.TrimSpace(pubkey)
			for k, v := range creds {
				if strings.HasSuffix(k, ".pub") {
					tk := client.AtsKey{}
					if err := json.Unmarshal(v, &tk); err != nil {
						return nil, err
					}
					if strings.TrimSpace(tk.KeyValue.Public) == pubkey {
						targets[k] = v
						pkname := strings.Replace(k, ".pub", ".sec", 1)
						targets[pkname] = creds[pkname]
						return targets, nil
					}
				}
			}
		}
	}
	return targets, errors.New("Unable to find offline target key for factory")
}
