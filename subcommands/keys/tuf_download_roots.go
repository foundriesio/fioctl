package keys

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"

	canonical "github.com/docker/go/canonical/json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	downloadRoots := &cobra.Command{
		Use:   "download-roots <archive path>",
		Short: "Download all versions of the Factory's TUF root metadata into a tarball",
		Run:   doDownloadRoots,
		Args:  cobra.ExactArgs(1),
	}
	downloadRoots.Flags().BoolP("prod", "", false, "Download the production versions")
	tufCmd.AddCommand(downloadRoots)
}

func doDownloadRoots(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	prod, _ := cmd.Flags().GetBool("prod")
	dstPath := args[0]

	var getRoot func(factory string, version int) (*client.AtsTufRoot, error)
	if prod {
		getRoot = api.TufProdRootGetVer
	} else {
		getRoot = api.TufRootGetVer
	}

	file, err := os.Create(dstPath)
	subcommands.DieNotNil(err)
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for ver := 1; ; ver++ {
		var root *client.AtsTufRoot
		root, err = getRoot(factory, ver)
		if err != nil {
			if herr := client.AsHttpError(err); herr != nil && herr.Response.StatusCode == 404 {
				break
			}
			subcommands.DieNotNil(err)
		}

		bytes, err := canonical.MarshalCanonical(root)
		subcommands.DieNotNil(err)

		fmt.Printf("= Adding %d.root.json\n", ver)
		header := &tar.Header{
			Name: fmt.Sprintf("%d.root.json", ver),
			Size: int64(len(bytes)),
			Mode: 0644,
		}
		subcommands.DieNotNil(tarWriter.WriteHeader(header))
		_, err = tarWriter.Write(bytes)
		subcommands.DieNotNil(err)
	}
}
