//go:build windows
// +build windows

package version

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Windows isn't able to update a running program like all the Unix
// platforms. We have to do a somewhat complex set of operations that
// can be destructive if things failed at the exact right moment:
//  1. Write new copy of fioctl. We'll call this fioctl-new.exe
//  2. Rename the existing fioctl. Call this fioctl-old.exe
//  3. Rename fioctl-new -> fioctl
//  4. Create a detached process
//     5 a) fioctl exits
//     5 b) child hack deletes original copy
func updateSelf(exe string, buff []byte, mode fs.FileMode) error {
	newFile := strings.Replace(exe, ".exe", "-new.exe", 1)
	oldFile := strings.Replace(exe, ".exe", "-old.exe", 1)

	if err := os.WriteFile(newFile, buff, mode); err != nil {
		return err
	}
	if err := os.Rename(exe, oldFile); err != nil {
		return err
	}
	if err := os.Rename(newFile, exe); err != nil {
		msg := "unable to update to the new fioctl binary. "
		msg += "The old version is located at %s. "
		msg += "The new version is located at %s. "
		return fmt.Errorf(msg, oldFile, newFile, err)
	}

	delSelfArgs := []string{"cmd.exe", "/C", "start", "/b", "timeout", "1", "&&", "del", oldFile}
	cmd := exec.Command(delSelfArgs[0], delSelfArgs[1:]...)
	devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0755)
	cmd.Stdout = devnull
	cmd.Stderr = devnull
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		msg := "the new version of fioctl is ready to use. "
		msg += "However, there was a failure removing the old version: %w"
		return fmt.Errorf(msg, err)
	}
	return nil
}
