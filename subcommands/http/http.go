package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var api *client.Api

func NewCommand() *cobra.Command {
	httpCmd := &cobra.Command{
		Use:    "http",
		Short:  "Run a direct authenticated HTTP command to the Foundries.io API",
		Hidden: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			api = subcommands.Login(cmd)
		},
	}
	httpCmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")

	getCmd := &cobra.Command{
		Use:   "get https://api.foundries.io/ota.... [header=val..]",
		Short: "Do an authenticated HTTP GET",
		Run:   doGet,
		Args:  cobra.MinimumNArgs(1),
	}
	httpCmd.AddCommand(getCmd)

	postCmd := &cobra.Command{
		Use:   "post https://api.foundries.io/ota.... [header=val..]",
		Short: "Do an authenticated HTTP POST",
		Run:   doPost,
		Args:  cobra.MinimumNArgs(1),
		Example: `# Post data directly from CLI:
fioctl post -d '{"key": "value"}' https://... content-type=application/json

# Post data from a file:
fioctl post -d @/tmp/tmp.json  https://... content-type=application/json

# Post data from STDIN:
echo '{"key": "value"}' | fioctl post -d - https://... content-type=application/json
`,
	}
	postCmd.Flags().StringP("data", "d", "", "HTTP POST data")
	httpCmd.AddCommand(postCmd)

	return httpCmd
}

func doGet(cmd *cobra.Command, args []string) {
	printResponse(api.RawGet(args[0], readHeaders(args[1:])))
}

func doPost(cmd *cobra.Command, args []string) {
	printResponse(api.RawPost(args[0], readData(cmd), readHeaders(args[1:])))
}

func readHeaders(args []string) *map[string]string {
	if len(args) == 0 {
		return nil
	}
	headers := make(map[string]string, len(args))
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 1 {
			headers[arg] = ""
		} else {
			headers[parts[0]] = parts[1]
		}
	}
	return &headers
}

func readData(cmd *cobra.Command) []byte {
	data, _ := cmd.Flags().GetString("data")
	if data == "-" {
		logrus.Debug("Reading post data from stdin")
		dataBytes, err := io.ReadAll(os.Stdin)
		subcommands.DieNotNil(err)
		return dataBytes
	} else if len(data) > 0 && data[0] == '@' {
		dataFile := data[1:]
		logrus.Debugf("Reading post data from %s", dataFile)
		dataBytes, err := os.ReadFile(dataFile)
		subcommands.DieNotNil(err)
		return dataBytes
	} else {
		return []byte(data)
	}
}

func printResponse(resp *http.Response, err error) {
	subcommands.DieNotNil(err)
	fmt.Fprintf(os.Stderr, "< Status: %s\n", resp.Status)
	for k, v := range resp.Header {
		fmt.Fprintf(os.Stderr, "< %s: %s\n", k, v)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	subcommands.DieNotNil(err)
}
