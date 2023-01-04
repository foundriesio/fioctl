package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const GIT_CREDS_HELPER = "git-credential-fio"

var (
	helperPath string
)

func NewCommand() *cobra.Command {
	_, err := exec.LookPath("git")

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
	cmd.Flags().StringVarP(&helperPath, "creds-path", "", helperPath, "Path to install credential helper")

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
	if len(helperPath) == 0 {
		path, err := getSymlinkDir()
		subcommands.DieNotNil(err)
		helperPath = path
	}

	self := findSelf()

	c := exec.Command("git", "config", "--global", "credential.https://source.foundries.io.username", "fio-oauth2")
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)
	c = exec.Command("git", "config", "--global", "credential.https://source.foundries.io.helper", "fio")
	out, err = c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)

	dst := filepath.Join(helperPath, GIT_CREDS_HELPER)
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
