//go:build !windows
// +build !windows

package version

import (
	"os"
)

func replaceSelf(curExe, newExe string) error {
	return os.Rename(newExe, curExe)
}
