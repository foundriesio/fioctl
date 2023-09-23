package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// FioCLIBinName defines the name of the binary for the tests
const FioCLIBinName = "fioctl"

// TestContext specified to run e2e tests
type TestContext struct {
	*CmdContext
	BinaryName string
}

// NewTestContext init a context for the tests
func NewTestContext(binaryName string, env ...string) (*TestContext, error) {
	cc := &CmdContext{
		Env: env,
	}

	return &TestContext{
		CmdContext: cc,
		BinaryName: binaryName,
	}, nil
}

// CmdContext provides context for command execution
type CmdContext struct {
	Env   []string
	Dir   string
	Stdin io.Reader
}

// Run executes the provided command within this context
func (cc *CmdContext) Run(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = cc.Dir
	cmd.Env = append(os.Environ(), cc.Env...)
	cmd.Stdin = cc.Stdin
	command := strings.Join(cmd.Args, " ")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// Ctl is for running the CLI commands
func (t *TestContext) Ctl(makeOptions ...string) (string, error) {
	cmd := exec.Command(t.BinaryName, makeOptions...)
	output, err := t.Run(cmd)
	return string(output), err
}
