package targets

import (
	"fmt"
	"os"
	"strconv"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "create-file <file name> <version> <hwid> <tag>",
		Short: "TMP hack",
		Run:   doCreateFile,
		Args:  cobra.ExactArgs(4),
	})
}

func doCreateFile(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	file := args[0]
	verStr := args[1]
	hwid := args[2]
	tag := args[3]
	version, err := strconv.Atoi(verStr)
	subcommands.DieNotNil(err)

	content, err := os.ReadFile(file)
	subcommands.DieNotNil(err)
	hash, err := api.CreateFile(factory, content)
	subcommands.DieNotNil(err)
	fmt.Println("Uploaded as:", hash)

	tgt := client.MCUTarget{
		Sha256:  hash,
		Version: version,
		HwId:    hwid,
		Tags:    []string{tag},
	}
	err = api.CreateMcuTarget(factory, tgt)
	subcommands.DieNotNil(err)
}
