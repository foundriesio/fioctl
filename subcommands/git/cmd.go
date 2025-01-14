package git

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const GIT_CREDS_HELPER = "git-credential-fio"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-git",
		Short: "Configure a source.foundries.io Git credential helper",
		Long: `Configure a Git credential helper that allows Git to access
source.foundries.io.

This command likely needs to be run as root. It creates a symlink,
git-credential-fio, in the same directory as the git client binary.

NOTE: The credentials will need the "source:read-update" scope to work with Git`,
		Run: doGitCreds,
		PreRun: func(cmd *cobra.Command, args []string) {
			_, err := exec.LookPath("git")
			subcommands.DieNotNil(err, "Git not found on system")
		},
	}
	return cmd
}

func NewGetCredsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-credential-helper",
		Hidden: true, // its used as a git-credential helper and is not user facing
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] != "get" {
				subcommands.DieNotNil(fmt.Errorf("This credential helper only supports 'get' and not '%s'", args[0]))
			}
			os.Exit(RunCredsHelper())
		},
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

	apiUrl := viper.GetString("server.url")
	if len(apiUrl) == 0 {
		apiUrl = "https://api.foundries.io"
	}
	parts, err := url.Parse(apiUrl)
	subcommands.DieNotNil(err)
	sourceUrl := strings.Replace(parts.Host, "api.", "source.", 1)

	cfgFile, err := filepath.Abs(viper.GetViper().ConfigFileUsed())
	subcommands.DieNotNil(err)

	if runtime.GOOS == "windows" {
		// To get around edge cases with git on Windows we use the absolute path
		// So for example the following path will be used: C:/Program\\ Files/Git/bin/git-credential-fio.exe
		self = strings.ReplaceAll(filepath.ToSlash(self), " ", "\\ ")
		cfgFile = strings.ReplaceAll(filepath.ToSlash(cfgFile), " ", "\\ ")
	}

	helper := fmt.Sprintf("%s git-credential-helper -c %s", self, cfgFile)
	gitUsernameCommandArgs := []string{"config", "--global", fmt.Sprintf("credential.https://%s.username", sourceUrl), "fio-oauth2"}
	gitHelperCommandArgs := []string{"config", "--global", fmt.Sprintf("credential.https://%s.helper", sourceUrl), helper}

	c := exec.Command("git", gitUsernameCommandArgs...)
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)
	c = exec.Command("git", gitHelperCommandArgs...)
	out, err = c.CombinedOutput()
	if len(out) > 0 {
		fmt.Printf("%s\n", string(out))
	}
	subcommands.DieNotNil(err)
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
