package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const GIT_CREDS_HELPER = "git-credential-fio"

var (
	helperPath string
)

func NewCommand() *cobra.Command {
	path, err := exec.LookPath("git")
	if err == nil {
		helperPath = filepath.Dir(path)
	}
	helperPath = findWritableDirInPath(helperPath)

	cmd := &cobra.Command{
		Use:   "configure-git",
		Short: "Configure a source.foundries.io Git credential helper",
		Long: `Configure a Git credential helper that allows Git to access
source.foundries.io.

This command will likely need to be run as root. It creates a symlink,
git-credential-fio, in the same directory as the git client binary.

NOTE: The credentials will need the "source:read-update" scope to work with Git`,
		Run: doGitCreds,
		PreRun: func(cmd *cobra.Command, args []string) {
			subcommands.DieNotNil(err, "Git not found on system")
		},
	}
	cmd.Flags().StringVarP(&helperPath, "creds-path", "", helperPath, "Path to install credential helper. This needs to be writable and in $PATH")
	if len(helperPath) == 0 {
		_ = cmd.MarkFlagRequired("creds-path")
	}
	return cmd
}

func findSelf() string {
	self := os.Args[0]
	if !filepath.IsAbs(self) {
		logrus.Debugf("Looking up path to %s", self)
		var err error
		self, err = exec.LookPath(self)
		subcommands.DieNotNil(err)
		self, err = filepath.Abs(self)
		subcommands.DieNotNil(err)
	}
	return filepath.Clean(self)
}

func doGitCreds(cmd *cobra.Command, args []string) {
	self := findSelf()

	sudoer := os.Getenv("SUDO_USER")
	var execCommand string
	var gitUsernameCommandArgs []string
	var gitHelperCommandArgs []string

	helperName := "fio"
	dst := filepath.Join(helperPath, GIT_CREDS_HELPER)
	if runtime.GOOS == "windows" {
		dst += ".exe"
		helperName += ".exe"
	}

	if len(sudoer) > 0 {
		u, err := user.Lookup(sudoer)
		subcommands.DieNotNil(err)
		execCommand = "su"
		gitUsernameCommandArgs = []string{u.Username, "-c", "git config --global credential.https://source.foundries.io.username fio-oauth2"}
		gitHelperCommandArgs = []string{u.Username, "-c", "git config --global credential.https://source.foundries.io.helper " + helperName}
	} else {
		execCommand = "git"
		gitUsernameCommandArgs = []string{"config", "--global", "credential.https://source.foundries.io.username", "fio-oauth2"}
		gitHelperCommandArgs = []string{"config", "--global", "credential.https://source.foundries.io.helper", helperName}
	}
	c := exec.Command(execCommand, gitUsernameCommandArgs...)
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)
	c = exec.Command(execCommand, gitHelperCommandArgs...)
	out, err = c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)

	fmt.Println("Symlinking", self, "to", dst)
	subcommands.DieNotNil(os.Symlink(self, dst))
}

func RunCredsHelper() int {
	if subcommands.Config.ClientCredentials.ClientSecret == "" {
		msg := "ERROR: Your fioctl configuration does not appear to include oauth2 credentials. Please run `fioctl login` to configure and then try again.\n"
		os.Stderr.WriteString(msg)
		os.Exit(1)
	}
	subcommands.Login(NewCommand()) // Ensure a fresh oauth2 access tokenA
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		subcommands.DieNotNil(err)
	}
	input += fmt.Sprintf("password=%s\n", subcommands.Config.ClientCredentials.AccessToken)
	os.Stdout.WriteString(input)
	return 0
}

// Find an entry in the PATH we can write to. For example, on MacOS git is
// installed under /usr/bin but even root can't write to that because of
// filesystem protection logic they have.
func findWritableDirInPath(gitPath string) string {
	path := os.Getenv("PATH")
	paths := make(map[string]bool)
	for _, part := range filepath.SplitList(path) {
		paths[part] = true
	}

	// Give preference to git location if its in PATH
	if len(gitPath) > 0 {
		if _, ok := paths[gitPath]; ok {
			if isWritable(gitPath) {
				return gitPath
			}
		}
	}

	// Now try everything
	for _, path := range filepath.SplitList(path) {
		if isWritable(path) {
			return path
		}
	}
	return ""
}
