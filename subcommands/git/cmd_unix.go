//go:build !windows
// +build !windows

package git

import "golang.org/x/sys/unix"

func isWritable(dir string) bool {
	return unix.Access(dir, unix.W_OK) == nil
}
