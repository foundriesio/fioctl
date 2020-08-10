package targets

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "image <target>",
		Short: "Generate a system image with pre-loaded container images",
		Run:   doImage,
		Args:  cobra.ExactArgs(1),
	})
}

func doImage(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	inputTarget := args[0]
	logrus.Debugf("Generating image of Target %s in Factory %s", inputTarget, factory)

	url, err := api.TargetImageCreate(factory, inputTarget)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("CI URL: %s\n", url)
}
