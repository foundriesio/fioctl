package targets

import (
	"archive/tar"
	"compress/bzip2"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type (
	ouTargetInfo struct {
		version       int
		ostreeVersion int
		hardwareID    string
		buildTag      string
	}
)

var (
	ouTag  string
	ouProd bool
)

func init() {
	offlineUpdateCmd := &cobra.Command{
		Use:   "offline-update <target-name> <dst> --tag <tag> [--prod]",
		Short: "Download Target content for an offline update",
		Run:   doOfflineUpdate,
		Args:  cobra.ExactArgs(2),
		Example: `
	# Download update content of the production target #1451 tagged by "release-01" for "intel-corei7-64" hardware type
	fioctl targets offline-update intel-corei7-64-lmp-1451 /mnt/flash-drive/offline-update-content --tag release-01 --prod

	# Download update content of the CI target #1451 tagged by "devel" for "raspberrypi4-64" hardware type
	fioctl targets raspberrypi4-64-lmp-1448 /mnt/flash-drive/offline-update-content --tag devel

	`,
	}
	cmd.AddCommand(offlineUpdateCmd)
	offlineUpdateCmd.Flags().StringVarP(&ouTag, "tag", "", "",
		"Target tag")
	offlineUpdateCmd.Flags().BoolVarP(&ouProd, "prod", "", false,
		"Instruct to fetch content of production Target")
}

func doOfflineUpdate(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	targetName := args[0]
	dstDir := args[1]

	if len(ouTag) == 0 {
		subcommands.DieNotNil(errors.New("missing mandatory flag `--tag`"))
	}

	fmt.Printf("Checking whether Target exists; target: %s, tag: %s, production: %v\n", targetName, ouTag, ouProd)
	subcommands.DieNotNil(checkIfTargetExists(factory, targetName, ouTag, ouProd))

	ti, err := getTargetInfo(factory, targetName)
	subcommands.DieNotNil(err, "Failed to obtain Target's details:")

	fmt.Printf("Downloading offline update content of Target %s to %s\n", targetName, dstDir)

	fmt.Println("Downloading TUF metadata...")
	subcommands.DieNotNil(downloadTufRepo(factory, ouTag, ouProd, path.Join(dstDir, "tuf")), "Failed to download TUF metadata:")

	fmt.Printf("Downloading an ostree repo from the Target's OE build %d...\n", ti.ostreeVersion)
	subcommands.DieNotNil(downloadOstree(factory, ti.ostreeVersion, ti.hardwareID, dstDir), "Failed to download Target's ostree repo:")

	fmt.Printf("Downloading Apps fetched by the `assemble-system-image` run; build number:  %d, tag: %s...\n", ti.version, ti.buildTag)
	err = downloadApps(factory, targetName, ti.version, ti.buildTag, path.Join(dstDir, "apps"))
	if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
		fmt.Println("WARNING: The Target Apps were not fetched by the `assemble` run, make sure that App preloading is enabled if needed. The update won't include any Apps!")
	} else {
		subcommands.DieNotNil(err, "Failed to download Target's Apps:")
	}

	fmt.Println("Successfully downloaded offline update content")
}

func checkIfTargetExists(factory string, targetName string, tag string, prod bool) error {
	data, err := api.TufMetadataGet(factory, "targets.json", tag, prod)
	if err != nil {
		if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
			return fmt.Errorf("the specified Target has not been found; target: %s, tag: %s, production: %v", targetName, ouTag, ouProd)
		}
		return fmt.Errorf("failed to check whether Target exists: %s", err.Error())
	}
	targets := client.AtsTufTargets{}
	err = json.Unmarshal(*data, &targets)
	if err != nil {
		return fmt.Errorf("failed to check whether Target exists: %s", err.Error())
	}
	for tn := range targets.Signed.Targets {
		if tn == targetName {
			return nil
		}
	}
	return fmt.Errorf("the specified Target has not been found; target: %s, tag: %s, production: %v", targetName, ouTag, ouProd)
}

func getTargetInfo(factory string, targetName string) (*ouTargetInfo, error) {
	// Getting Target's custom info from the `/ota/factories/<factory>/targets/<target-name>` because:
	// 1. a target name is unique and represents the same Target across all "tagged" targets set including prod;
	// 2. only this target version/representation contains an original tag(s)/branch that
	// the `image-assemble` and apps fetching was performed for (needed for determining where to download Apps from).
	custom, err := getTargetCustomInfo(factory, targetName)
	if err != nil {
		return nil, err
	}

	info := ouTargetInfo{}
	info.version, err = strconv.Atoi(custom.Version)
	if err != nil {
		return nil, err
	}
	info.hardwareID = custom.HardwareIds[0]
	info.buildTag = custom.Tags[0] // See the assemble.py script in ci-scripts https://github.com/foundriesio/ci-scripts/blob/18b4fb154c37b6ad1bc6e7b7903a540b7a758f5d/assemble.py#L300
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

func downloadTufRepo(factory string, tag string, prod bool, dstDir string) error {
	// v1 - auto-generated by tuf_keyserver (default, on Factory creation);
	// v2 - auto-generated by ota-lite to take keys online (default, on Factory creation);
	// v3 - customer takes keys offline (prerequisite for offline update);
	ver := 4

	downloadMetadataFile := func(metadataFileName string) error {
		data, err := api.TufMetadataGet(factory, metadataFileName, tag, prod)
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

	metadataFileNames := []string{
		"timestamp.json", "snapshot.json", "targets.json",
	}
	for _, file := range metadataFileNames {
		err = downloadMetadataFile(file)
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

func getTargetCustomInfo(factory string, targetName string) (*client.TufCustom, error) {
	targetFile, err := api.TargetGet(factory, targetName)
	if err != nil {
		return nil, err
	}
	custom, err := api.TargetCustom(*targetFile)
	return custom, err
}

func downloadItem(factory string, targetVer int, runName string, artifactPath string, storeHandler func(r io.Reader) error) error {
	resp, err := api.JobservRunArtifact(factory, targetVer, runName, artifactPath)
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
