package version

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
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
	cmd.Flags().StringVarP(&updateTo, "update-to", "", "", "Update to the given version of Fioctl")
	return cmd
}

func doVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Commit)
	updateFinder := newUpdateFinder(cmd)
	update, err := updateFinder.FindLatest()
	DieNotNil(err)

	if len(updateTo) != 0 {
		if update == nil {
			DieNotNil(fmt.Errorf("no updates found for platform: %s", updateFinder.platform))
			return // required to make compiler happy
		}
		if updateTo != update.Version.String() {
			DieNotNil(fmt.Errorf("invalid version %s != %s", updateTo, update.Version))
		}
		if err := update.Do(); err != nil {
			if os.IsPermission(err) {
				DieNotNil(fmt.Errorf("There was a permission error while updating Fioctl: %w. Please run again as an admin or root.", err))
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
		fmt.Println("This is the latest version.")
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

// We must define this here are face circular import issues
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
