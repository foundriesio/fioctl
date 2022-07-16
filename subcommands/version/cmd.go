package version

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Commit string

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information of this tool.",
		Run:   doVersion,
	}
}

func doVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Commit)
	updateFinder := newUpdateFinder(cmd)
	update, err := updateFinder.FindLatest()
	DieNotNil(err)
	if update != nil {
		fmt.Println("Update available:", update.Version)
		fmt.Println("\t", update.Uri)
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
