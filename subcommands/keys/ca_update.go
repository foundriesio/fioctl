package keys

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

func init() {
	cmd := &cobra.Command{
		Use:   "update <ca-crts file>",
		Short: "Update the list of CAs that can create client certificates for devices",
		Run:   doUpdateCA,
		Args:  cobra.ExactArgs(1),
	}
	caCmd.AddCommand(cmd)
}

func doUpdateCA(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Updating certs for %s", factory)

	buf, err := ioutil.ReadFile(args[0])
	subcommands.DieNotNil(err)

	certs := client.CaCerts{CaCrt: string(buf)}
	subcommands.DieNotNil(api.FactoryPatchCA(factory, certs))
}
