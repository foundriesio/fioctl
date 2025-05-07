package targets

import (
	"archive/tar"
	"compress/bzip2"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	canonical "github.com/docker/go/canonical/json"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/foundriesio/fioctl/subcommands/keys"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type (
	ouTargetInfo struct {
		version       int
		ostreeVersion int
		hardwareID    string
		buildTag      string
		fetchedApps   *client.FetchedApps
		sha256        []byte
	}

	ouBundleMeta struct {
		Type    string   `json:"type"`
		Tag     string   `json:"tag"`
		Targets []string `json:"targets"`
	}
	ouBundleTufMeta struct {
		tuf.SignedCommon
		ouBundleMeta `json:"x-fio-offline-bundle"`
	}
)

var (
	ouTag                  string
	ouProd                 bool
	ouExpiresIn            int
	ouTufOnly              bool
	ouNoApps               bool
	ouAllowMultipleTargets bool
	ouWave                 string
	ouOstreeRepoSrc        string
)

func init() {
	offlineUpdateCmd := &cobra.Command{
		Use:   "offline-update <target-name> <dst> --tag <tag> [--prod | --wave <wave-name>] [--expires-in-days <days>] [--tuf-only]",
		Short: "Download Target content for an offline update",
		Run:   doOfflineUpdate,
		Args:  cobra.ExactArgs(2),
		Example: `
	# Download update content of the Wave Target #1451 for "intel-corei7-64" hardware type
	fioctl targets offline-update intel-corei7-64-lmp-1451 /mnt/flash-drive/offline-update-content --wave wave-deployment-001

	# Download update content of the production Target #1451 tagged by "release-01" for "intel-corei7-64" hardware type
	fioctl targets offline-update intel-corei7-64-lmp-1451 /mnt/flash-drive/offline-update-content --tag release-01 --prod

	# Download update content of the CI Target #1451 tagged by "devel" for "raspberrypi4-64" hardware type
	fioctl targets offline-update raspberrypi4-64-lmp-1448 /mnt/flash-drive/offline-update-content --tag devel --expires-in-days 15

	`,
	}
	cmd.AddCommand(offlineUpdateCmd)
	offlineUpdateCmd.Flags().StringVarP(&ouTag, "tag", "", "",
		"Target tag")
	offlineUpdateCmd.Flags().BoolVarP(&ouProd, "prod", "", false,
		"Instruct to fetch content of production Target")
	offlineUpdateCmd.Flags().StringVarP(&ouWave, "wave", "", "",
		"Instruct to fetch content of Wave Target; a wave name should be specified")
	offlineUpdateCmd.Flags().IntVarP(&ouExpiresIn, "expires-in-days", "e", 30,
		"Desired metadata validity period in days")
	offlineUpdateCmd.Flags().BoolVarP(&ouTufOnly, "tuf-only", "m", false,
		"Fetch only TUF metadata")
	offlineUpdateCmd.Flags().BoolVarP(&ouNoApps, "no-apps", "", false,
		"Skip fetching Target Apps")
	offlineUpdateCmd.Flags().BoolVarP(&ouAllowMultipleTargets, "allow-multiple-targets", "", false,
		"Allow multiple Targets to be stored in the same <dst> directory")
	offlineUpdateCmd.Flags().StringVarP(&ouOstreeRepoSrc, "ostree-repo-source", "", "",
		"Path to the local ostree repo to be added to the offline bundle")
	offlineUpdateCmd.MarkFlagsMutuallyExclusive("tag", "wave")
	offlineUpdateCmd.MarkFlagsMutuallyExclusive("prod", "wave")
	initSignCmd(offlineUpdateCmd)
	initShowCmd(offlineUpdateCmd)
}

func initSignCmd(parentCmd *cobra.Command) {
	signCmd := &cobra.Command{
		Use:   "sign <path to an offline bundle>",
		Short: "Sign an offline bundle with a Targets role offline key",
		Long: `Sign an offline bundle with a Targets role offline key.

Run this command if your offline update bundle contains production/Wave targets.
In this case, the bundle has to be signed by one or more Targets role offline key.
The number of required signatures depends on the threshold number set in the current TUF root role metadata,
and is printed by this command.`,
		Run:  doSignBundle,
		Args: cobra.ExactArgs(1),
	}
	signCmd.Flags().StringP("keys", "k", "",
		"Path to the <tuf-targets-keys.tgz> key to sign the bundle metadata with. "+
			"This is the same key used to sign prod & Wave TUF Targets.")
	_ = signCmd.MarkFlagRequired("keys")
	parentCmd.AddCommand(signCmd)
}

func initShowCmd(parentCmd *cobra.Command) {
	showCmd := &cobra.Command{
		Use:   "show <path to an offline bundle>",
		Short: "Parse and print the specified bundle metadata",
		Long: `Parse and print the specified bundle metadata.

Run this command if you would like to get information about an offline bundle.
Specifically, what Targets it includes, what the type of the Targets (CI or production),
a bundle's expiration time', etc.`,
		Run:  doShowBundle,
		Args: cobra.ExactArgs(1),
	}
	parentCmd.AddCommand(showCmd)
}

func doOfflineUpdate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	targetName := args[0]
	dstDir := args[1]

	if len(ouTag) == 0 && len(ouWave) == 0 {
		subcommands.DieNotNil(errors.New("Either `--tag` or `--wave` should be specified"))
	}

	var targetCustomData *tuf.FileMeta
	var targetGetErr error
	// Get the wave/prod/CI specific target with the specified tag to check if it is really present
	if len(ouWave) > 0 {
		fmt.Printf("Getting Wave Target details; target: %s, wave: %s...\n", targetName, ouWave)
		targetCustomData, targetGetErr = getWaveTargetMeta(factory, targetName, ouWave)
	} else if ouProd {
		fmt.Printf("Getting production Target details; target: %s, tag: %s...\n", targetName, ouTag)
		targetCustomData, targetGetErr = getProdTargetMeta(factory, targetName, ouTag)
	} else {
		fmt.Printf("Getting CI Target details; target: %s, tag: %s...\n", targetName, ouTag)
		targetCustomData, targetGetErr = getCiTargetMeta(factory, targetName, ouTag)
	}
	subcommands.DieNotNil(targetGetErr)

	fmt.Printf("Refreshing and downloading TUF metadata for Target %s to %s...\n", targetName, path.Join(dstDir, "tuf"))
	subcommands.DieNotNil(downloadTufRepo(factory, targetName, ouTag, ouProd, ouWave, ouExpiresIn, path.Join(dstDir, "tuf")), "Failed to download TUF metadata:")
	fmt.Println("Successfully refreshed and downloaded TUF metadata")

	if !ouTufOnly {
		if !isDstDirClean(dstDir) {
			if !ouAllowMultipleTargets {
				subcommands.DieNotNil(errors.New(`Destination directory already has update data.
Provide a clean destination directory or re-run with --allow-multiple-targets to add a new Target to a directory which already has update data.
Notice that multiple Targets in the same directory is only supported in LmP >= v92.`))
			}
		}

		// Get the target info in order to deduce the ostree and app download URLs
		ti, err := getTargetInfo(targetCustomData)
		subcommands.DieNotNil(err)

		if ouOstreeRepoSrc != "" {
			fmt.Printf("Copying local ostree repo from %s...\n", ouOstreeRepoSrc)
			subcommands.DieNotNil(copyOstree(ouOstreeRepoSrc, dstDir+"/ostree_repo/"), "Failed to copy local ostree repo:")
		} else {
			fmt.Printf("Downloading an ostree repo from the Target's OE build %d...\n", ti.ostreeVersion)
			subcommands.DieNotNil(downloadOstree(factory, ti.ostreeVersion, ti.hardwareID, dstDir), "Failed to download Target's ostree repo:")
		}

		sha256String := base64.StdEncoding.EncodeToString(ti.sha256)
		expectedCommit := path.Join(dstDir, "ostree_repo", "objects", sha256String[0:2], sha256String[2:]+".commit")
		if _, err := os.Stat(expectedCommit); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("ERROR: ostree repo does not contain target's hash %s. If the target references a custom ostree repo, re-run specifying --ostree-repo-source.\n", sha256String)
			os.Exit(1)
		}

		if !ouNoApps {
			if (len(ouWave) > 0 || ouProd) && ti.fetchedApps == nil {
				// Get the specified target from the list of factory targets to obtain the "original" tag/branch that produced
				// the target, so we can find out the correct app bundle fetch URL.
				targetCustomData, targetGetErr = api.TargetGet(factory, targetName)
				subcommands.DieNotNil(targetGetErr)

				// Get the target info again in order to extract the "original" tag/branch and deduce the app download URLs
				ti, err = getTargetInfo(targetCustomData)
				subcommands.DieNotNil(err)
			}

			if ti.fetchedApps == nil {
				fmt.Printf("Downloading Apps fetched by the `assemble-system-image` run; build number: %d, tag: %s...\n", ti.version, ti.buildTag)
				err = downloadApps(factory, targetName, ti.version, ti.buildTag, path.Join(dstDir, "apps"))
			} else {
				if len(ti.fetchedApps.Uri) > 0 {
					fmt.Printf("Downloading Apps fetched by the `publish-compose-apps` run; apps: %s, uri: %s...\n", ti.fetchedApps.Shortlist, ti.fetchedApps.Uri)
					err = downloadAppsArchive(ti.fetchedApps.Uri, path.Join(dstDir, "apps"))
				} else {
					fmt.Println("No apps found to fetch for an offline update to a given Target. " +
						"The bundle will only update rootfs/ostree. Check your Factory configuration if this is not your intention.")
				}
			}
			if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
				fmt.Println("WARNING: The Target Apps were not fetched by the `assemble` run, make sure that App preloading is enabled if needed. The update won't include any Apps!")
			} else {
				subcommands.DieNotNil(err, "Failed to download Target's Apps:")
			}
		}
		fmt.Println("Successfully downloaded offline update content")
	}
	doShowBundle(cmd, []string{dstDir})
}

func getTargetInfo(targetFile *tuf.FileMeta) (*ouTargetInfo, error) {
	// Since the wave/prod/CI specific target json usually doesn't contain the "original" branch/tag that the apps were fetched for
	// we do the following to determine where to fetch the target app bundle from.
	// Getting Target's custom info from the `/ota/factories/<factory>/targets/<target-name>` because:
	// 1. a target name is unique and represents the same Target across all "tagged" targets set including prod;
	// 2. only this target version/representation contains an original tag(s)/branch that
	// the `image-assemble` and apps fetching was performed for (needed for determining where to download Apps from).
	custom, err := api.TargetCustom(*targetFile)
	if err != nil {
		return nil, err
	}
	info := ouTargetInfo{}
	info.version, err = strconv.Atoi(custom.Version)
	if err != nil {
		return nil, err
	}
	info.hardwareID = custom.HardwareIds[0]
	info.fetchedApps = custom.FetchedApps
	sha256, ok := targetFile.Hashes["sha256"]
	if ok {
		info.sha256 = sha256
	} else {
		fmt.Printf("ERROR: Target has no sha256 hash set")
		os.Exit(1)
	}
	if info.fetchedApps == nil {
		info.buildTag = custom.Tags[0] // See the assemble.py script in ci-scripts https://github.com/foundriesio/ci-scripts/blob/18b4fb154c37b6ad1bc6e7b7903a540b7a758f5d/assemble.py#L300
	}
	info.ostreeVersion = info.version
	if len(custom.OrigUri) > 0 {
		indx := strings.LastIndexByte(custom.OrigUri, '/')
		if indx == -1 {
			return nil, fmt.Errorf("failed to determine Target's OE build version: %s", custom.OrigUri)
		}
		targetOstreeVerStr := custom.OrigUri[indx+1:]
		info.ostreeVersion, err = strconv.Atoi(targetOstreeVerStr)
		if err != nil {
			return nil, err
		}
	}
	return &info, nil
}

func getWaveTargetMeta(factory string, targetName string, wave string) (*tuf.FileMeta, error) {
	waveTargets, err := api.WaveTargetsList(factory, true, wave)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return nil, fmt.Errorf("No active Wave with the specified name was found; wave: %s", wave)
		}
		return nil, fmt.Errorf("Failed to get Wave Target metadata: %s", err.Error())
	}
	if foundTargetMeta, ok := waveTargets[wave].Signed.Targets[targetName]; ok {
		return &foundTargetMeta, nil
	} else {
		return nil, fmt.Errorf("The specified Target is not found among Wave Targets;"+
			" target: %s, wave: %s", targetName, wave)
	}
}

func getProdTargetMeta(factory string, targetName string, tag string) (*tuf.FileMeta, error) {
	targets, err := api.ProdTargetsGet(factory, tag, true)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return nil, fmt.Errorf("No production Targets were found for the specified tag `%s`", tag)
		}
		return nil, fmt.Errorf("Failed to get production Target metadata: %s", err.Error())
	}
	if foundTargetMeta, ok := targets.Signed.Targets[targetName]; ok {
		return &foundTargetMeta, nil
	} else {
		return nil, fmt.Errorf("No production Target with the given tag found;"+
			" target: %s, tag: %s", targetName, tag)
	}
}

func getCiTargetMeta(factory string, targetName string, tag string) (*tuf.FileMeta, error) {
	data, err := api.TufMetadataGet(factory, "targets.json", tag, false)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return nil, fmt.Errorf("No CI Targets found for the specified tag `%s`", tag)
		}
		return nil, fmt.Errorf("Failed to get CI Target metadata: %s", err.Error())
	}
	targets := client.AtsTufTargets{}
	err = json.Unmarshal(*data, &targets)
	if err != nil {
		return nil, err
	}
	if foundTargetMeta, ok := targets.Signed.Targets[targetName]; ok {
		return &foundTargetMeta, nil
	} else {
		return nil, fmt.Errorf("No CI Target with the given tag found;"+
			" target: %s, tag: %s", targetName, tag)
	}
}

func isDstDirClean(dstDir string) bool {
	for _, subDir := range []string{"ostree_repo", "apps"} {
		fullPath := path.Join(dstDir, subDir)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			fmt.Println(fullPath + " already exists")
			return false
		}
	}
	return true
}

func downloadTufRepo(factory string, target string, tag string, prod bool, wave string, expiresIn int, dstDir string) error {
	// v1 - auto-generated by tuf_keyserver (default, on Factory creation);
	// v2 - auto-generated by ota-lite to take keys online (default, on Factory creation);
	ver := 3
	downloadMetadataFile := func(metadataFileName string) error {
		data, err := api.TufMetadataGet(factory, metadataFileName, tag, prod || len(wave) > 0)
		if err != nil {
			return err
		}
		f, err := os.Create(path.Join(dstDir, metadataFileName))
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(*data)
		return err
	}

	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		return err
	}

	for {
		metadataFileName := fmt.Sprintf("%d.root.json", ver)
		err := downloadMetadataFile(metadataFileName)
		if err != nil {
			if httpErr := client.AsHttpError(err); httpErr != nil && httpErr.Response.StatusCode == 404 {
				// if 404 received for N.root.json, then stop downloading root metadata versions
				break
			}
			return err
		}
		ver += 1
	}
	bundleTargets, err := getBundleTargetsMeta(dstDir, false)
	if err != nil {
		return err
	}
	meta, err := api.TufTargetMetadataRefresh(factory, target, tag, expiresIn, prod, wave, bundleTargets)
	if err != nil {
		return err
	}
	metadataNames := []string{
		"timestamp", "snapshot", "targets", "bundle-targets",
	}
	for _, metaName := range metadataNames {
		b, err := canonical.MarshalCanonical(meta[metaName])
		if err != nil {
			return err
		}
		err = os.WriteFile(path.Join(dstDir, metaName+".json"), b, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyRecursive(src, dst string) error {
	copyFile := func(src, dst string) error {
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			return nil
		}

		srcFile, err := os.Open(src)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		tmpDstName := dst + ".tmp"
		dstFile, err := os.Create(tmpDstName)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = dstFile.ReadFrom(srcFile)
		if err != nil {
			return err
		}
		err = dstFile.Sync()
		if err != nil {
			return err
		}
		err = os.Rename(tmpDstName, dst)
		if err != nil {
			return err
		}

		return err
	}

	return filepath.Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstPath := path.Join(dst, strings.Replace(srcPath, src, "", 1))
		if info.IsDir() {
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				return os.Mkdir(dstPath, info.Mode())
			} else {
				return nil
			}
		} else {
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
			return os.Chmod(dstPath, info.Mode())
		}
	})
}

func copyOstree(srcDir string, dstDir string) error {
	return copyRecursive(srcDir, dstDir)
}

func downloadOstree(factory string, targetVer int, hardwareID string, dstDir string) error {
	runName := hardwareID
	artifactName := hardwareID + "-ostree_repo.tar.bz2"
	artifactPath := path.Join("other", artifactName)

	return downloadItem(factory, targetVer, runName, artifactPath, func(r io.Reader) error {
		bzr := bzip2.NewReader(r)
		if bzr == nil {
			return fmt.Errorf("failed to create bzip2 reader")
		}
		return untar(bzr, dstDir)
	})
}

func downloadApps(factory string, targetName string, targetVer int, tag string, dstDir string) error {
	runName := "assemble-system-image"
	artifactPath := path.Join(tag, targetName+"-apps.tar")

	return downloadItem(factory, targetVer, runName, artifactPath, func(r io.Reader) error {
		return untar(r, dstDir)
	})
}

func downloadAppsArchive(uri string, dstDir string) error {
	resp, err := api.RawGet(uri, nil)
	return processDownloadResponse(uri, resp, err, func(r io.Reader) error {
		return untar(r, dstDir)
	})
}

func downloadItem(factory string, targetVer int, runName string, artifactPath string, storeHandler func(r io.Reader) error) error {
	resp, err := api.JobservRunArtifact(factory, targetVer, runName, artifactPath)
	return processDownloadResponse(artifactPath, resp, err, storeHandler)
}

func processDownloadResponse(artifactPath string, resp *http.Response, err error, storeHandler func(r io.Reader) error) error {
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &client.HttpError{
			Message:  fmt.Sprintf("failed to download a CI artifact; status code: %d, artifact: %s", resp.StatusCode, artifactPath),
			Response: resp,
		}
	}

	status := DlStatus{resp.ContentLength, 0, 20, time.Now()}
	return storeHandler(io.TeeReader(resp.Body, &status))
}

func untar(r io.Reader, dstDir string) error {
	tr := tar.NewReader(r)
	storeItem := func(flag byte, name string, size int64) error {
		switch flag {
		case tar.TypeDir:
			return os.MkdirAll(path.Join(dstDir, name), 0755)
		default:
			f, err := os.Create(path.Join(dstDir, name))
			if err != nil {
				return err
			}
			defer f.Close()
			w, err := io.Copy(f, tr)
			if err != nil {
				return err
			}
			if w != size {
				return err
			}
		}
		return nil
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		err = storeItem(hdr.Typeflag, hdr.Name, hdr.Size)
		if err != nil {
			return err
		}
	}

	return nil
}

func getBundleTargetsMeta(bundleTufPath string, errIfNotExist bool) (bundleTargets *tuf.Signed, err error) {
	if b, readErr := os.ReadFile(path.Join(bundleTufPath, "bundle-targets.json")); readErr == nil {
		var foundBundleTargets tuf.Signed
		if err = canonical.Unmarshal(b, &foundBundleTargets); err == nil {
			bundleTargets = &foundBundleTargets
		}
	} else if errIfNotExist || !errors.Is(readErr, os.ErrNotExist) {
		err = readErr
	}
	return
}

func doSignBundle(cmd *cobra.Command, args []string) {
	offlineKeysFile, _ := cmd.Flags().GetString("keys")
	offlineKeys, err := keys.GetOfflineCreds(offlineKeysFile)
	subcommands.DieNotNil(err, "Failed to open offline keys file")

	bundleTufPath := path.Join(args[0], "tuf")
	bundleTargets, err := getBundleTargetsMeta(bundleTufPath, true)
	subcommands.DieNotNil(err)

	rootMeta, err := getLatestRoot(bundleTufPath)
	subcommands.DieNotNil(err)

	err = signBundleTargets(rootMeta, bundleTargets, offlineKeys)
	subcommands.DieNotNil(err)

	if b, err := canonical.MarshalCanonical(bundleTargets); err == nil {
		subcommands.DieNotNil(os.WriteFile(path.Join(bundleTufPath, "bundle-targets.json"), b, 0666))
		numberOfMoreRequiredSignatures := rootMeta.Signed.Roles["targets"].Threshold - len(bundleTargets.Signatures)
		if numberOfMoreRequiredSignatures > 0 {
			fmt.Printf("%d more signature(s) is/are required to meet the required threshold (%d)\n",
				numberOfMoreRequiredSignatures, rootMeta.Signed.Roles["targets"].Threshold)
		} else {
			fmt.Printf("The bundle is signed with enough number of signatures (%d) to meet the required threshold (%d)\n",
				len(bundleTargets.Signatures), rootMeta.Signed.Roles["targets"].Threshold)
		}
	} else {
		subcommands.DieNotNil(err)
	}
	doShowBundle(cmd, args)
}

func getLatestRoot(bundleTufPath string) (*client.AtsTufRoot, error) {
	var latestVersionBytes []byte
	var readErr error

	curVer := 3
	for {

		latestVersionPath := path.Join(bundleTufPath, fmt.Sprintf("%d.root.json", curVer))
		if b, err := os.ReadFile(latestVersionPath); err != nil {
			readErr = err
			break
		} else {
			latestVersionBytes = b
			curVer += 1
		}
	}
	if !errors.Is(readErr, os.ErrNotExist) {
		return nil, readErr
	}
	if latestVersionBytes == nil {
		// None of the N.root.json where N starts from 3 was found in the bundle
		return nil, os.ErrNotExist
	}
	rootMeta := client.AtsTufRoot{}
	if err := json.Unmarshal(latestVersionBytes, &rootMeta); err != nil {
		return nil, err
	}
	return &rootMeta, nil
}

func signBundleTargets(rootMeta *client.AtsTufRoot, bundleTargetsMeta *tuf.Signed, offlineKeys keys.OfflineCreds) error {
	signer, err := keys.FindOneTufSigner(rootMeta, offlineKeys, rootMeta.Signed.Roles["targets"].KeyIDs)
	if err != nil {
		return fmt.Errorf("%s %w", keys.ErrMsgReadingTufKey("targets", "current"), err)
	}
	for _, signature := range bundleTargetsMeta.Signatures {
		if signature.KeyID == signer.Id {
			return fmt.Errorf("the bundle is already signed by the provided key: %s", signer.Id)
		}
	}
	if bundleTargetsMeta.Signed == nil {
		panic(fmt.Errorf("the input bundle metadata to sign is nil"))
	}
	fmt.Printf("Signing the bundle with new key; ID: %s, type: %s\n", signer.Id, signer.Type.Name())
	signatures, err := keys.SignTufMeta(*bundleTargetsMeta.Signed, signer)
	if err != nil {
		return err
	}
	bundleTargetsMeta.Signatures = append(bundleTargetsMeta.Signatures, signatures[0])
	return nil
}

func doShowBundle(cmd *cobra.Command, args []string) {
	tufMetaPath := path.Join(args[0], "tuf")
	bundleTufMeta, err := getBundleTargetsMeta(tufMetaPath, true)
	subcommands.DieNotNil(err)
	bundleMeta := ouBundleTufMeta{}
	subcommands.DieNotNil(json.Unmarshal(*bundleTufMeta.Signed, &bundleMeta))
	fmt.Println("Bundle Targets info:")
	fmt.Printf("\tType:\t\t%s\n", bundleMeta.ouBundleMeta.Type)
	fmt.Printf("\tTag:\t\t%s\n", bundleMeta.Tag)
	fmt.Printf("\tExpires:\t%s\n", bundleMeta.Expires)
	fmt.Println("\tTargets:")
	for _, target := range bundleMeta.Targets {
		fmt.Printf("\t\t\t%s\n", target)
	}
	fmt.Println("\tSignatures:")
	for _, sig := range bundleTufMeta.Signatures {
		fmt.Printf("\t\t\t- %s\n", sig.KeyID)
	}

	rootMeta, err := getLatestRoot(tufMetaPath)
	if errors.Is(err, os.ErrNotExist) && bundleMeta.ouBundleMeta.Type == "ci" {
		// If no any N.root.json is found in the bundle and this is the "ci" bundle,
		// then this is the valid case - a user has not taken their TUF targets key offline.
		// Therefore, instead of failing the command fetches the root meta from the backend.
		rootMeta, err = api.TufRootGet(viper.GetString("factory"))
	}
	subcommands.DieNotNil(err)
	fmt.Println("\tAllowed keys:")
	for _, key := range rootMeta.Signed.Roles["targets"].KeyIDs {
		fmt.Printf("\t\t\t- %s\n", key)
	}
	fmt.Printf("\tThreshold:\t%d\n", rootMeta.Signed.Roles["targets"].Threshold)
	numberOfMissingSignatures := rootMeta.Signed.Roles["targets"].Threshold - len(bundleTufMeta.Signatures)
	if numberOfMissingSignatures > 0 {
		fmt.Printf("\tMissing:\t%d (the number of required additional signatures;"+
			" run the `sign` sub-command to sign the bundle)\n", numberOfMissingSignatures)
	}
}
