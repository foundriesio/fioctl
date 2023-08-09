//go:build !windows
// +build !windows

package version

import (
	"io/fs"
	"os"
)

func updateSelf(exe string, buff []byte, mode fs.FileMode) error {
	tmp := exe + ".tmp"
	if err := os.WriteFile(tmp, buff, mode); err != nil {
		return err
	}
	return os.Rename(tmp, exe)
}
