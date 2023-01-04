//go:build unix

package git

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func getSymlinkDir() (string, error) {
	candidates := make([]string, 0)
	if path, ok := os.LookupEnv("PATH"); ok {
		paths := strings.Split(path, ":")
		for _, p := range paths {
			// Checks if directory is writable
			if err := syscall.Access(p, syscall.O_RDWR); err == nil {
				candidates = append(candidates, p)
			}
		}
	}
	if len(candidates) == 0 {
		fmt.Println("No writable directory found.")
		fmt.Println("Please add a directory that is writable to your PATH.")
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
