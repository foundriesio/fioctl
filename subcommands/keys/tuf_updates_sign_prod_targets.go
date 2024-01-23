package keys

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	signCmd := &cobra.Command{
		Use:   "sign-prod-targets --txid=<txid> --keys=<tuf-root-keys.tgz>",
		Short: "Sign production targets for your Factory with the offline targets key",
		Long: `Sign production targets for your Factory with the offline targets key.
New signatures are staged for commit along with TUF root modifications.

There are 3 use cases when this command comes handy:

- You want to sign your Factory's production targets with a newly added offline TUF Targets key.
- You increase the TUF targets signature threshold
  and need to sign your production targets with an additional key.
- You remove an offline TUF targets keys
  and need to replace its signatures on production targets with signatures by another key.`,
		Run: doTufUpdatesSignProdTargets,
	}
	signCmd.Flags().StringP("txid", "x", "", "TUF root updates transaction ID.")
	signCmd.Flags().StringP("keys", "k", "", "Path to <tuf-targets-keys.tgz> used to sign TUF targets.")
	_ = signCmd.MarkFlagFilename("keys")
	_ = signCmd.MarkFlagRequired("keys")
	signCmd.Flags().StringP("tags", "", "", "A comma-separated list of tags to sign; default: all tags.")
	signCmd.Flags().StringP("waves", "", "", "A comma-separated list of waves to sign; default: all active waves.")
	tufUpdatesCmd.AddCommand(signCmd)
}

func doTufUpdatesSignProdTargets(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	txid, _ := cmd.Flags().GetString("txid")
	keysFile, _ := cmd.Flags().GetString("keys")
	tagsStr, _ := cmd.Flags().GetString("tags")
	wavesStr, _ := cmd.Flags().GetString("waves")
	var tags, waveNames []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	if wavesStr != "" {
		waveNames = strings.Split(wavesStr, ",")
	}

	creds, err := GetOfflineCreds(keysFile)
	subcommands.DieNotNil(err)

	updates, err := api.TufRootUpdatesGet(factory)
	subcommands.DieNotNil(err)

	_, newCiRoot, newProdRoot := checkTufRootUpdatesStatus(updates, true)
	if newProdRoot == nil {
		subcommands.DieNotNil(errors.New(`Please, make changes to your Factory TUF root.
For example, add a new offline TUF targets key, before signing production targets with it.`))
	}

	onlineTargetsId := updates.Updated.OnlineKeys["targets"]
	if onlineTargetsId == "" {
		subcommands.DieNotNil(errors.New("Unable to find online target key for factory"))
	}
	signer, err := FindOneTufSigner(newCiRoot, creds,
		subcommands.SliceRemove(newCiRoot.Signed.Roles["targets"].KeyIDs, onlineTargetsId))
	subcommands.DieNotNil(err, ErrMsgReadingTufKey(tufRoleNameTargets, "current"))

	var newTargetsProdSigs, newTargetsWaveSigs map[string][]tuf.Signature

	fmt.Println("= Signing prod targets")
	// If both wave names and tags specified, or none specified - re-sign both prod and wave targets.
	// If only wave names or only tags specified - re-sign only what was specified (either wave names or tags).
	if len(tags) > 0 || len(waveNames) == 0 {
		targetsProdMap, err := api.ProdTargetsList(factory, true, tags...)
		subcommands.DieNotNil(err, "Failed to fetch production targets:")
		newTargetsProdSigs, err = signProdTargets(signer, targetsProdMap)
		subcommands.DieNotNil(err)
	}
	if len(waveNames) > 0 || len(tags) == 0 {
		targetsWaveMap, err := api.WaveTargetsList(factory, true, waveNames...)
		subcommands.DieNotNil(err, "Failed to fetch production wave targets:")
		newTargetsWaveSigs, err = signProdTargets(signer, targetsWaveMap)
		subcommands.DieNotNil(err)
	}

	fmt.Println("= Uploading new signatures")
	subcommands.DieNotNil(api.TufRootUpdatesPut(
		factory, txid, newCiRoot, newProdRoot, newTargetsProdSigs, newTargetsWaveSigs))
}
