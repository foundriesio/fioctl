package subcommands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewPostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "post https://api.foundries.io/ota.... [header=val..]",
		Short:  "Do an authenticated HTTP POST",
		Hidden: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = Login(cmd)
		},
		Run:  doPost,
		Args: cobra.MinimumNArgs(1),
	}
	cmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
	cmd.Flags().StringP("data", "d", "", "HTTP POST data")
	return cmd
}

func doPost(cmd *cobra.Command, args []string) {
	headers := make(map[string]string)

	if len(args) > 1 {
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 1 {
				headers[arg] = ""
			} else {
				headers[parts[0]] = parts[1]
			}
		}
	}

	data, _ := cmd.Flags().GetString("data")

	resp, err := api.RawPost(args[0], []byte(data), &headers)
	DieNotNil(err)
	fmt.Fprintf(os.Stderr, "< Status: %s\n", resp.Status)
	for k, v := range resp.Header {
		fmt.Fprintf(os.Stderr, "< %s: %s\n", k, v)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	DieNotNil(err)
}
