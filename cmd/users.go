package cmd

import (
	"fmt"
	"os"

	"github.com/cheynewallace/tabby"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var usersCmd = &cobra.Command{
	Use:              "users",
	Short:            "List users with access to a factory",
	PersistentPreRun: assertLogin,
	Run:              doUsersList,
}

func init() {
	rootCmd.AddCommand(usersCmd)
	requireFactory(usersCmd)
}

func doUsersList(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Listing factory users for %s", factory)

	users, err := api.UsersList(factory)
	if err != nil {
		fmt.Print("ERROR: ")
		fmt.Println(err)
		os.Exit(1)
	}

	t := tabby.New()
	t.AddHeader("ID", "NAME", "ROLE")
	for _, user := range users {
		t.AddLine(user.PolisId, user.Name, user.Role)
	}
	t.Print()
}
