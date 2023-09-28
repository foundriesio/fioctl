package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const DOCKER_CREDS_HELPER = "docker-credential-fio"

var (
	dockerConfigFile string
	helperPath       string
)

func NewCommand() *cobra.Command {
	path, err := exec.LookPath("docker")
	if err == nil {
		helperPath = filepath.Dir(path)
	}
	helperPath = subcommands.FindWritableDirInPath(helperPath)
	dockerConfigFile = dockerConfigPath()
	if !subcommands.IsWritable(dockerConfigFile) {
		dockerConfigFile = ""
	}

	cmd := &cobra.Command{
		Use:   "configure-docker",
		Short: "Configure a hub.foundries.io Docker credential helper",
		Long: `Configure a Docker credential helper that allows Docker to access
hub.foundries.io.

This command will likely need to be run as root. It creates a symlink,
docker-credential-fio, in the same directory as the docker client binary.

NOTE: The credentials will need the "containers:read" scope to work with Docker`,
		Run: doDockerCreds,
		PreRun: func(cmd *cobra.Command, args []string) {
			subcommands.DieNotNil(err, "Docker not found on system")
		},
	}
	cmd.Flags().StringVarP(&helperPath, "creds-path", "", helperPath, "Path to install credential helper")
	if len(helperPath) == 0 {
		_ = cmd.MarkFlagRequired("creds-path")
	}
	cmd.Flags().StringVarP(&dockerConfigFile, "docker-config", "", dockerConfigFile, "Docker config file to update")
	if len(dockerConfigFile) == 0 {
		_ = cmd.MarkFlagRequired("docker-config")
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

func dockerConfigPath() string {
	sudoer := os.Getenv("SUDO_USER")
	if len(sudoer) > 0 {
		u, err := user.Lookup(sudoer)
		subcommands.DieNotNil(err)
		return filepath.Join(u.HomeDir, ".docker/config.json")
	}
	path, err := homedir.Expand("~/.docker/config.json")
	subcommands.DieNotNil(err)
	return path
}

func doDockerCreds(cmd *cobra.Command, args []string) {
	self := findSelf()

	var config map[string]interface{}
	bytes, err := os.ReadFile(dockerConfigFile)
	if errors.Is(err, fs.ErrNotExist) {
		dockerConfig := filepath.Dir(dockerConfigFile)
		if _, err := os.Stat(dockerConfig); errors.Is(err, fs.ErrNotExist) {
			fmt.Println("Creating docker config directory:", dockerConfig)
			subcommands.DieNotNil(os.Mkdir(dockerConfig, 0o755))
		}
		config = make(map[string]interface{})
	} else {
		subcommands.DieNotNil(json.Unmarshal(bytes, &config))
	}

	helpers, ok := config["credHelpers"]
	if !ok {
		config["credHelpers"] = map[string]string{
			"hub.foundries.io": "fio",
		}
	} else {
		helpers.(map[string]interface{})["hub.foundries.io"] = "fio"
	}

	configBytes, err := subcommands.MarshalIndent(config, "", "  ")
	subcommands.DieNotNil(err)

	dst := filepath.Join(helperPath, DOCKER_CREDS_HELPER)
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}
	fmt.Println("Symlinking", self, "to", dst)
	subcommands.DieNotNil(os.Symlink(self, dst))

	fmt.Println("Adding hub.foundries.io helper to", dockerConfigFile)
	subcommands.DieNotNil(os.WriteFile(dockerConfigFile, configBytes, 0o600))
}

func RunCredsHelper() int {
	if subcommands.Config.ClientCredentials.ClientSecret == "" {
		msg := "ERROR: Your fioctl configuration does not appear to include oauth2 credentials. " +
			"Please run `fioctl login` to configure and then try again.\n"
		os.Stderr.WriteString(msg)
		os.Exit(1)
	}
	subcommands.Login(NewCommand()) // Ensure a fresh oauth2 access token
	creds := struct {
		Username string
		Secret   string
	}{
		Username: "<token>",
		Secret:   subcommands.Config.ClientCredentials.AccessToken,
	}

	bytes, err := json.Marshal(creds)
	subcommands.DieNotNil(err)
	os.Stdout.Write(bytes)
	return 0
}
