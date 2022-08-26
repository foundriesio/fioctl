package el2g

import (
	"fmt"
	"strconv"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the overall status of the Edgelock2Go integration",
		Run:   doStatus,
	})
}

func doStatus(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	products, err := api.El2gProducts(factory)
	subcommands.DieNotNil(err)

	overview, err := api.El2gOverview(factory)
	subcommands.DieNotNil(err)
	fmt.Println("# Subdomain:", overview.Subdomain)
	fmt.Println("\n# Product IDs")
	t := subcommands.Tabby(1, "ID", "NAME")
	for _, id := range overview.ProductIds {
		name := ""
		for _, prod := range products {
			if prod.Nc12 == strconv.Itoa(id) {
				name = prod.Type
				break
			}
		}
		t.AddLine(id, name)
	}
	t.Print()

	secureObjects, err := api.El2gSecureObjects(factory)
	subcommands.DieNotNil(err)
	fmt.Println("\n# Secure Objects")
	t = subcommands.Tabby(1, "TYPE", "NAME", "OBJECT ID")
	for _, so := range secureObjects {
		t.AddLine(so.Type, so.Name, so.ObjectId)
	}
	t.Print()

	fmt.Println("\n# Intermediate CAs")
	cas, err := api.El2gIntermediateCas(factory)
	subcommands.DieNotNil(err)
	for i, ca := range cas {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println("Name:", ca.Name)
		fmt.Println("Algorithm:", ca.Algorithm)
		fmt.Println("ID:", ca.Id)
		fmt.Println(ca.Value)
	}
}
