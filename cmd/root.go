package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	cfgcmd "github.com/foundriesio/fioctl/subcommands/config"
	"github.com/foundriesio/fioctl/subcommands/devices"
	"github.com/foundriesio/fioctl/subcommands/docker"
	"github.com/foundriesio/fioctl/subcommands/events"
	"github.com/foundriesio/fioctl/subcommands/factories"
	"github.com/foundriesio/fioctl/subcommands/git"
	"github.com/foundriesio/fioctl/subcommands/http"
	"github.com/foundriesio/fioctl/subcommands/keys"
	"github.com/foundriesio/fioctl/subcommands/login"
	"github.com/foundriesio/fioctl/subcommands/logout"
	"github.com/foundriesio/fioctl/subcommands/secrets"
	"github.com/foundriesio/fioctl/subcommands/status"
	"github.com/foundriesio/fioctl/subcommands/targets"
	"github.com/foundriesio/fioctl/subcommands/teams"
	"github.com/foundriesio/fioctl/subcommands/users"
	"github.com/foundriesio/fioctl/subcommands/version"
	"github.com/foundriesio/fioctl/subcommands/waves"
)

var (
	cfgFile string
	config  client.Config
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:               "fioctl",
	Short:             "Manage Foundries Factories",
	PersistentPreRunE: rootArgValidation,
}

func Execute() {
	if strings.Contains(os.Args[0], docker.DOCKER_CREDS_HELPER) {
		if len(os.Args) != 2 || os.Args[1] != "get" {
			fmt.Printf("Usage: %s get\n", os.Args[0])
			os.Exit(1)
		}
		initConfig()
		os.Exit(docker.RunCredsHelper())
	}
	if strings.Contains(os.Args[0], git.GIT_CREDS_HELPER) {
		if len(os.Args) != 2 || os.Args[1] != "get" {
			// It is either store or erase, we do not do anything on both
			return
		}
		initConfig()
		os.Exit(git.RunCredsHelper())
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.EnableTraverseRunHooks = true
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/fioctl.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print verbose logging")

	rootCmd.AddCommand(completionCmd)

	rootCmd.AddCommand(cfgcmd.NewCommand())
	rootCmd.AddCommand(devices.NewCommand())
	rootCmd.AddCommand(docker.NewCommand())
	rootCmd.AddCommand(git.NewCommand())
	rootCmd.AddCommand(git.NewGetCredsCommand())
	rootCmd.AddCommand(events.NewCommand())
	rootCmd.AddCommand(factories.NewCommand())
	rootCmd.AddCommand(keys.NewCommand())
	rootCmd.AddCommand(login.NewCommand())
	rootCmd.AddCommand(logout.NewCommand())
	rootCmd.AddCommand(users.NewCommand())
	rootCmd.AddCommand(teams.NewCommand())
	rootCmd.AddCommand(secrets.NewCommand())
	rootCmd.AddCommand(status.NewCommand())
	rootCmd.AddCommand(targets.NewCommand())
	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(waves.NewCommand())
	rootCmd.AddCommand(http.NewCommand())
	rootCmd.AddCommand(&cobra.Command{Use: "get", Hidden: true, Deprecated: "Use 'http get' instead"})
	rootCmd.AddCommand(&cobra.Command{Use: "post", Hidden: true, Deprecated: "Use 'http post' instead"})

	rootCmd.AddCommand(docsRstCmd)
	rootCmd.AddCommand(docsMdCmd)
}

func rootArgValidation(cmd *cobra.Command, args []string) error {
	if strings.HasPrefix(cmd.Name(), "__complete") {
		return nil
	}
	for pos, val := range args {
		if len(strings.TrimSpace(val)) == 0 {
			return fmt.Errorf("Empty values or values containing only white space are not allowed for positional argument at %d\n", pos)
		}
	}
	return nil
}

func getConfigDir() string {
	config, err := homedir.Expand("~/.config")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := os.Stat(config); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(config, 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return config
}

func initConfig() {
	cfgFileFromEnv := os.Getenv("FIOCTL_CONFIG")
	if len(cfgFileFromEnv) > 0 {
		cfgFile = cfgFileFromEnv
	}
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name "fioctl" (without extension).
		viper.AddConfigPath(getConfigDir())
		viper.SetConfigName("fioctl")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("FIOCTL")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Debug("Config file not found")
		} else {
			// Config file was found but another error was produced
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	}
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Sprintf("Unexpected failure parsing configuration: %s", err))
	}
	subcommands.Config = config
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|powershell]",
	Short: "Generate completion script",
	Example: `
# Bash:
$ source <(fioctl completion bash)

# To load completions for each session, execute once:
Linux:
  $ fioctl completion bash > /etc/bash_completion.d/fioctl
MacOS:
  $ fioctl completion bash > /usr/local/etc/bash_completion.d/fioctl

# Zsh:
# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ fioctl completion zsh > "${fpath[1]}/_fioctl"

# You will need to start a new shell for this setup to take effect.

# Fish:
$ fioctl completion fish > ~/.config/fish/completions/fioctl.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				logrus.Fatal(err)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				logrus.Fatal(err)
			}
		case "fish":
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				logrus.Fatal(err)
			}
		case "powershell":
			if err := cmd.Root().GenPowerShellCompletion(os.Stdout); err != nil {
				logrus.Fatal(err)
			}
		}
	},
}
