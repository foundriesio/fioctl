package targets

import (
	"fmt"
	"os"
	"regexp"

	"github.com/fatih/color"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

var (
	appsShortlist string
	noTail        bool
	ciScriptsRepo string
	ciScriptsRef  string
)

func init() {
	var imageCmd = &cobra.Command{
		Use:   "image <target-name>",
		Short: "Generate a system image with pre-loaded container images",
		Example: "fioctl targets image raspberrypi4-64-lmp-464 // preload all Target apps\n" +
			"fioctl targets image raspberrypi4-64-lmp-464 --apps app-00,app-01 // preload app-00 and app-01",
		Run:  doImage,
		Args: cobra.ExactArgs(1),
	}
	cmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&appsShortlist, "apps", "", "",
		"comma,separate,list of Target apps to preload into a resultant image."+
			" All apps of Target are preloaded if the flag is not defined or empty")
	imageCmd.Flags().BoolVarP(&noTail, "no-tail", "", false, "Don't tail output of CI Job")
	imageCmd.Flags().StringVarP(&ciScriptsRepo, "ci-scripts-repo", "", "https://github.com/foundriesio/ci-scripts", "Override to custom version of ci-scripts")
	imageCmd.Flags().StringVarP(&ciScriptsRef, "ci-scripts-ref", "", "master", "Override to a specific git-ref of ci-scripts")
}

func validateAppShortlist() {
	pattern := `^[a-zA-Z0-9-_,]+$`
	re := regexp.MustCompile(pattern)
	if len(appsShortlist) > 0 && !re.MatchString(appsShortlist) {
		color.Red("ERROR: Invalid value for apps:", appsShortlist)
		color.Red("       apps must be ", pattern)
		os.Exit(1)
	}
}

func doImage(cmd *cobra.Command, args []string) {
	validateAppShortlist()
	factory := viper.GetString("factory")
	inputTarget := args[0]
	logrus.Debugf("Generating image of Target %s in Factory %s", inputTarget, factory)

	jobServUrl, webUrl, err := api.TargetImageCreate(factory, inputTarget, appsShortlist, ciScriptsRepo, ciScriptsRef)
	subcommands.DieNotNil(err)
	fmt.Printf("CI URL: %s\n", webUrl)
	if !noTail {
		api.JobservTail(jobServUrl)
	}
}
