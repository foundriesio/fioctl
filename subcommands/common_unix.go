//go:build !windows
// +build !windows

package subcommands

import "golang.org/x/sys/unix"

func IsWritable(dir string) bool {
	return unix.Access(dir, unix.W_OK) == nil
}
