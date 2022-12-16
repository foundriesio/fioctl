package targets

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var (
	addTargetType     string
	addTags           string
	addSrcTag         string
	addQuiet          bool
	addDryRun         bool
	addTargetsCreator string
)

type Targets map[string]*client.Target

func init() {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Compose and add Targets to Factory's  TUF targets metadata",
		Run:   doAdd,
		Long: `
Compose new Targets out of the latest Targets tagged with the specified source tag and the specified via the command arguments eiter OSTree commit hashes or App URIs.

fioctl targets add --type <ostree | app> --tags <comma,separate,list of Target tags> --src-tag <source Target tag> [--targets-creator <something about Targets originator>]\ 
	<hardware ID> <ostree commit hash> [<hardware ID> <ostree commit hash>]  (for ostree type)
	<App #1 URI> [App #N URI] (for app type)`,
		Example: `
Add new ostree Targets: 
	fioctl targets add --type ostree --tags dev,test --src-tag dev --targets-creator "custom jenkins ostree build" intel-corei7-64 00b2ad4a1dd7fe1e856a6d607ed492c354a423be22a44bad644092bb275e12fa raspberrypi4-64 5e05a59529dcdd54310945b2628d73c0533097d76cc483334925a901845b3794
		
Add new App Targets:
	fioctl targets add --type app --tags dev,test --src-tag dev hub.foundries.io/factory/simpleapp@sha256:be955ad958ef37bcce5afaaad32a21b783b3cc29ec3096a76484242afc333e26 hub.foundries.io/factory/app-03@sha256:59b080fe42d7c45bc81ea17ab772fc8b3bb5ef0950f74669d069a2e6dc266a24 
		`,
	}
	cmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addTargetType, "type", "", "", "Target type")
	addCmd.Flags().StringVarP(&addTags, "tags", "", "", "comma,separate,list of Target tags")
	addCmd.Flags().StringVarP(&addSrcTag, "src-tag", "", "", "OSTree Target tag to base app targets on")
	addCmd.Flags().BoolVarP(&addQuiet, "quiet", "", false, "don't print generated new Targets to stdout")
	addCmd.Flags().BoolVarP(&addDryRun, "dry-run", "", false, "don't post generated new Targets")
	addCmd.Flags().StringVarP(&addTargetsCreator, "targets-creator", "", "fioctl", "optional name/comment/context about Targets origination")
}

func doAdd(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	supportedTargetTypes := map[string]func(factory string, tags []string, srcTag string, args []string) (Targets, error){
		"app":    createAppTargets,
		"ostree": createOstreeTarget,
	}
	if len(addTargetType) == 0 {
		subcommands.DieNotNil(errors.New("missing mandatory flag `--type`"))
	}
	deriveTargets, ok := supportedTargetTypes[addTargetType]
	if !ok {
		subcommands.DieNotNil(errors.New("unsupported type of Target: " + addTargetType))
	}
	if len(addTags) == 0 {
		subcommands.DieNotNil(errors.New("missing mandatory flag `--tags`"))
	}
	tags := strings.Split(addTags, ",")
	if len(addSrcTag) == 0 {
		subcommands.DieNotNil(errors.New("missing mandatory flag `--src-tag`"))
	}

	newTargets, err := deriveTargets(factory, tags, addSrcTag, args)
	subcommands.DieNotNil(err)
	if !addQuiet {
		newTargets.print()
	}
	if !addDryRun {
		fmt.Printf("Posting new Targets...")
		err = newTargets.post(factory, addTargetsCreator)
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println("Failed")
		}
		subcommands.DieNotNil(err)
	}
}

func createAppTargets(factory string, tags []string, srcTag string, appUris []string) (Targets, error) {
	if len(appUris) == 0 {
		return nil, fmt.Errorf("at least one App URI is required as an argument")
	}
	newTargetApps := map[string]client.ComposeApp{}
	for _, u := range appUris {
		// TODO: check App URIs, make sure that it's actually a valid URI
		composeApp := client.ComposeApp{Uri: u}
		appName := composeApp.Name()
		newTargetApps[appName] = composeApp
	}
	// TBC:
	// 1. Parse App manifests to determine supported platforms/archs, map it to hardware IDs and pass them as an input param to `deriveTargets`
	// 2. Add CLI param `hw-ids` to shortlist by hardware IDs when new App Targets are being added,
	// instead of generating Targets for each hardware ID of the specified source tag.
	return deriveTargets(factory, nil, srcTag, func(target *client.Target) error {
		target.Custom.Tags = tags
		target.Custom.ComposeApps = newTargetApps
		return nil
	})
}

func createOstreeTarget(factory string, tags []string, srcTag string, hwIdToHashPairs []string) (Targets, error) {
	if len(hwIdToHashPairs) < 2 {
		return nil, fmt.Errorf("at least one pair `hw-id ostree-hash` is required as an argument")
	}
	if len(hwIdToHashPairs)%2 != 0 {
		return nil, fmt.Errorf("invalid pairs of `hw-id ostree-hash` are specified as an argument")
	}
	hwIdToHash := map[string]interface{}{}
	var curHwId string
	for ind, v := range hwIdToHashPairs {
		if ind%2 == 0 {
			curHwId = v
		} else {
			if _, ok := hwIdToHash[curHwId]; ok {
				return nil, fmt.Errorf("the same hardware ID is specified twice: %s", curHwId)
			}
			// TODO: make sure that `v` is actually sha256 hash
			hwIdToHash[curHwId] = v
		}
	}
	return deriveTargets(factory, hwIdToHash, srcTag, func(target *client.Target) error {
		target.Custom.Tags = tags
		err := target.SetHash(hwIdToHash[target.HardwareId()].(string))
		return err
	})
}

func deriveTargets(factory string, hwIds map[string]interface{}, srcTag string, customizeFunc func(target *client.Target) error) (Targets, error) {
	latestBuild, err := api.JobservLatestBuild(factory, false)
	subcommands.DieNotNil(err)

	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)

	latestTargetsPerHwId := make(map[string]*client.Target) // latest Targets per hardware ID that have  `srcTag`
	for _, meta := range targets {
		target, err := api.NewTarget(meta)
		if err != nil {
			return nil, err
		}
		curVer := target.Version()
		if !target.HasTag(srcTag) {
			continue
		}
		curTargetHwId := target.HardwareId()
		if hwIds != nil {
			if _, ok := hwIds[curTargetHwId]; !ok {
				// The current Target hw ID is not in the list of hardware IDs to add Targets for
				continue
			}
		}
		if latestTargetForSrcTag, exists := latestTargetsPerHwId[curTargetHwId]; !exists || latestTargetForSrcTag.Version() < curVer {
			latestTargetsPerHwId[curTargetHwId] = target
		}
	}
	// We assume that the initial OSTree Target(s) is/are created by the OTA service during Factory creation,
	// hence we don't need to create a new Target from scratch in this command, just derive Targets from existing one.
	if len(latestTargetsPerHwId) == 0 {
		return nil, fmt.Errorf("no any source Targets to derive new Targets from are found; source tag: %s", srcTag)
	}
	for id := range hwIds {
		if _, ok := latestTargetsPerHwId[id]; !ok {
			return nil, fmt.Errorf("no source Target to derive new Target from is found"+
				" for the source tag `%s` and the hardware ID `%s`", srcTag, id)
		}
	}
	newTargets := Targets{}
	fmt.Println("Deriving new Targets...")
	for _, latest := range latestTargetsPerHwId {
		fmt.Printf("\t %s -> ", latest.Name())
		newTarget := latest.DeriveTarget(latestBuild.ID + 1)
		if err := customizeFunc(newTarget); err != nil {
			return nil, err
		}
		newTargets[newTarget.Name()] = newTarget
		fmt.Printf("%s\n", newTarget.Name())
	}
	return newTargets, nil
}

func (t Targets) post(factory string, targetsCreator string) error {
	reqPayload := map[string]interface{}{
		"targets":         t,
		"targets-creator": targetsCreator,
	}
	b, err := json.Marshal(reqPayload)
	if err != nil {
		return err
	}
	return api.TargetsPost(factory, b)
}

func (t Targets) print() {
	b, err := subcommands.MarshalIndent(t, "", "\t")
	if err != nil {
		fmt.Printf("Failed to marshal new Targets: %s\n", err)
	}
	fmt.Printf("New Targets\n%s\n", string(b))
}
