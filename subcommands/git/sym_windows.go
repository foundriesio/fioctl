package git

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// https://learn.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants
const FILE_APPEND_FILE = 0x00000002

// https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew#parameters at dwShareMode
const FILE_LOCK = 0x00000000

func getSymlinkDir() (string, error) {
	candidates := make([]string, 0)
	if path, ok := os.LookupEnv("PATH"); ok {
		paths := strings.Split(path, ";")
		for _, p := range paths {
			// Checks if directory is writable
			if hwnd, err := syscall.CreateFile(syscall.StringToUTF16Ptr(p), FILE_APPEND_FILE, FILE_LOCK, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0); err == nil {
				candidates = append(candidates, p)
				if err = syscall.CloseHandle(hwnd); err != nil {
					return "", err
				}
			}
		}
	}
	if len(candidates) == 0 {
		fmt.Println("No writable directory found.")
		fmt.Println("Please add a directory that is writable to your PATH.")
		fmt.Println("More information here: https://stackoverflow.com/a/72341522")
		os.Exit(1)
	}
	fmt.Printf("\nPlease make a choice of which path the symlink will be placed:\n\n")
	for i, p := range candidates {
		fmt.Printf("[%d]: %s\n", i, p)
	}
	fmt.Printf("[Q]: Quit\n")

	var choiceIdx string
	_, err := fmt.Scanln(&choiceIdx)
	if err != nil {
		return "", nil
	}
	if strings.ToLower(choiceIdx) == "q" {
		os.Exit(0)
	}
	choice, err := strconv.Atoi(choiceIdx)
	if err != nil {
		return "", err
	}
	return candidates[choice], nil
}
