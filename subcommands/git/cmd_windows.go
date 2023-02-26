//go:build windows
// +build windows

package git

import (
	"syscall"
)

func isWritable(dir string) bool {
	if hwnd, err := syscall.CreateFile(syscall.StringToUTF16Ptr(dir), 2, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0); err == nil {
		_ = syscall.CloseHandle(hwnd)
		return true
	}
	return false
}
