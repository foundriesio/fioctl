package version

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Commit string
var updateTo string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information of this tool.",
		Run:   doVersion,
	}
	cmd.Flags().StringVarP(&updateTo, "update-to", "", "", "Update to the given version of fioctl")
	return cmd
}

func doVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Commit)
	updateFinder := newUpdateFinder(cmd)
	update, err := updateFinder.FindLatest()
	DieNotNil(err)

	if len(updateTo) != 0 {
		if updateTo != update.Version.String() {
			DieNotNil(fmt.Errorf("invalid version %s != %s", updateTo, update.Version))
		}
		if err := update.Do(); err != nil {
			if os.IsPermission(err) {
				fmt.Println("There was a permission error while updating fioctl. Please run this again with 'sudo'")
			} else {
				DieNotNil(err)
			}
		}
		return
	}

	if update != nil {
		fmt.Println("Update available:", update.Version)
		fmt.Println("\t", update.Uri)
		exe, err := os.Executable()
		DieNotNil(err)
		fmt.Println()
		fmt.Println("You can update by running:")
		fmt.Println("\t", exe, "version", "--update-to", update.Version)
	} else {
		fmt.Println("This is the latest version of the tool.")
	}
}

func newUpdateFinder(cmd *cobra.Command) *FioctlUpdateFinder {
	DieNotNil(viper.BindPFlags(cmd.Flags()))
	path, err := homedir.Expand("~/.config/fioctl-tuf.json")
	DieNotNil(err)
	client, err := NewFioctlUpdateFinder(path)
	DieNotNil(err)
	return client
}

func DieNotNil(err error, message ...string) {
	if err != nil {
		parts := []interface{}{"ERROR:"}
		for _, p := range message {
			parts = append(parts, p)
		}
		parts = append(parts, err)
		fmt.Println(parts...)
		os.Exit(1)
	}
}
