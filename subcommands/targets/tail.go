package targets

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "tail <target> <run>",
		Short: "Tail the console output of a live CI Run",
		Run:   doTail,
		Args:  cobra.ExactArgs(2),
		Example: `
  fioctl targets tail 12 build-amd64
`,
	})
}

func doTail(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	build, err := strconv.Atoi(args[0])
	subcommands.DieNotNil(err)
	run := args[1]

	api.JobservTailRun(factory, build, run, "console.log")
}
