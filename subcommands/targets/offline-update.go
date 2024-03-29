package targets

import (
	"archive/tar"
	"compress/bzip2"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
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
)

func init() {
	offlineUpdateCmd := &cobra.Command{
		Use:   "offline-update <target-name> <dst> --tag <tag> [--prod | --wave <wave-name>] [--expires-in-days <days>] [--tuf-only]",
		Short: "Download Target content for an offline update",
		Run:   doOfflineUpdate,
		Args:  cobra.ExactArgs(2),
		Example: `
	# Download update content of the wave target #1451 for "intel-corei7-64" hardware type
	fioctl targets offline-update intel-corei7-64-lmp-1451 /mnt/flash-drive/offline-update-content --wave wave-deployment-001

	# Download update content of the production target #1451 tagged by "release-01" for "intel-corei7-64" hardware type
	fioctl targets offline-update intel-corei7-64-lmp-1451 /mnt/flash-drive/offline-update-content --tag release-01 --prod

	# Download update content of the CI target #1451 tagged by "devel" for "raspberrypi4-64" hardware type
	fioctl targets offline-update raspberrypi4-64-lmp-1448 /mnt/flash-drive/offline-update-content --tag devel --expires-in-days 15

	`,
	}
	cmd.AddCommand(offlineUpdateCmd)
	offlineUpdateCmd.Flags().StringVarP(&ouTag, "tag", "", "",
		"Target tag")
	offlineUpdateCmd.Flags().BoolVarP(&ouProd, "prod", "", false,
		"Instruct to fetch content of production Target")
	offlineUpdateCmd.Flags().StringVarP(&ouWave, "wave", "", "",
		"Instruct to fetch content of wave Target; a wave name should be specified")
	offlineUpdateCmd.Flags().IntVarP(&ouExpiresIn, "expires-in-days", "e", 30,
		"Desired metadata validity period in days")
	offlineUpdateCmd.Flags().BoolVarP(&ouTufOnly, "tuf-only", "m", false,
		"Fetch only TUF metadata")
	offlineUpdateCmd.Flags().BoolVarP(&ouNoApps, "no-apps", "", false,
		"Skip fetching Target Apps")
	offlineUpdateCmd.Flags().BoolVarP(&ouAllowMultipleTargets, "allow-multiple-targets", "", false,
		"Allow multiple targets to be stored in the same <dst> directory")
	offlineUpdateCmd.MarkFlagsMutuallyExclusive("tag", "wave")
	offlineUpdateCmd.MarkFlagsMutuallyExclusive("prod", "wave")
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
Provide a clean destination directory or re-run with --allow-multiple-targets to add a new target to a directory which already has update data.
Notice that multiple targets in the same directory is only supported in LmP >= v92.`))
			}
		}

		// Get the target info in order to deduce the ostree and app download URLs
		ti, err := getTargetInfo(targetCustomData)
		subcommands.DieNotNil(err)

		fmt.Printf("Downloading an ostree repo from the Target's OE build %d...\n", ti.ostreeVersion)
		subcommands.DieNotNil(downloadOstree(factory, ti.ostreeVersion, ti.hardwareID, dstDir), "Failed to download Target's ostree repo:")
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
				fmt.Printf("Downloading Apps fetched by the `publish-compose-apps` run; apps: %s, uri: %s...\n", ti.fetchedApps.Shortlist, ti.fetchedApps.Uri)
				err = downloadAppsArchive(ti.fetchedApps.Uri, path.Join(dstDir, "apps"))
			}
			if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
				fmt.Println("WARNING: The Target Apps were not fetched by the `assemble` run, make sure that App preloading is enabled if needed. The update won't include any Apps!")
			} else {
				subcommands.DieNotNil(err, "Failed to download Target's Apps:")
			}
		}
		fmt.Println("Successfully downloaded offline update content")
	}
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
		return nil, fmt.Errorf("The specified Target is not found among wave targets;"+
			" target: %s, wave: %s", targetName, wave)
	}
}

func getProdTargetMeta(factory string, targetName string, tag string) (*tuf.FileMeta, error) {
	targets, err := api.ProdTargetsGet(factory, tag, true)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return nil, fmt.Errorf("No production targets were found for the specified tag `%s`", tag)
		}
		return nil, fmt.Errorf("Failed to get production Target metadata: %s", err.Error())
	}
	if foundTargetMeta, ok := targets.Signed.Targets[targetName]; ok {
		return &foundTargetMeta, nil
	} else {
		return nil, fmt.Errorf("No production target with the given tag is found;"+
			" target: %s, tag: %s", targetName, tag)
	}
}

func getCiTargetMeta(factory string, targetName string, tag string) (*tuf.FileMeta, error) {
	data, err := api.TufMetadataGet(factory, "targets.json", tag, false)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return nil, fmt.Errorf("No CI targets were found for the specified tag `%s`", tag)
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
		return nil, fmt.Errorf("No CI target with the given tag is found;"+
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

	meta, err := api.TufTargetMetadataRefresh(factory, target, tag, expiresIn, prod, wave)
	if err != nil {
		return err
	}
	metadataNames := []string{
		"timestamp", "snapshot", "targets",
	}
	for _, metaName := range metadataNames {
		b, err := json.Marshal(meta[metaName])
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
