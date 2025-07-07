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
			follow, _ := cmd.Flags().GetBool("follow-redirects")
			if !follow {
				api.GetHttpClient().CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}
			}
		},
	}
	httpCmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")

	getCmd := &cobra.Command{
		Use: "get https://api.foundries.io/ota.... [header=val..]",
		Run: doGet,
	}

	postCmd := &cobra.Command{
		Use: "post https://api.foundries.io/ota.... [header=val..]",
		Run: doPost,
	}

	putCmd := &cobra.Command{
		Use: "put https://api.foundries.io/ota.... [header=val..]",
		Run: doPut,
	}

	patchCmd := &cobra.Command{
		Use: "patch https://api.foundries.io/ota.... [header=val..]",
		Run: doPatch,
	}

	deleteCmd := &cobra.Command{
		Use: "delete https://api.foundries.io/ota.... [header=val..]",
		Run: doDelete,
	}

	for _, cmd := range []*cobra.Command{getCmd, postCmd, putCmd, patchCmd, deleteCmd} {
		cmd.Short = fmt.Sprintf("Do an authenticated HTTP %s", cmd.Name())
		cmd.Args = cobra.MinimumNArgs(1)
		httpCmd.AddCommand(cmd)
	}

	for _, cmd := range []*cobra.Command{postCmd, putCmd, patchCmd, deleteCmd} {
		cmd.Example = fmt.Sprintf(`# Do HTTP %s with data directly form CLI:
fioctl %s -d '{"key": "value"}' https://...

# Do HTTP %s with data from a file:
fioctl %s -d @/tmp/tmp.json  https://...

# Do HTTP %s with data from STDIN:
echo '{"key": "value"}' | fioctl %s -d - https://...
`, cmd.Name(), cmd.Name(), cmd.Name(), cmd.Name(), cmd.Name(), cmd.Name())
		cmd.Flags().StringP("data", "d", "", "HTTP POST data")
		cmd.Flags().BoolP("follow-redirects", "f", true, "Follow HTTP redirects")
	}

	return httpCmd
}

func doGet(cmd *cobra.Command, args []string) {
	printResponse(api.RawGet(args[0], readHeaders(args[1:])))
}

func doPost(cmd *cobra.Command, args []string) {
	printResponse(api.RawPost(args[0], readData(cmd), readHeaders(args[1:])))
}

func doPut(cmd *cobra.Command, args []string) {
	printResponse(api.RawPut(args[0], readData(cmd), readHeaders(args[1:])))
}

func doPatch(cmd *cobra.Command, args []string) {
	printResponse(api.RawPatch(args[0], readData(cmd), readHeaders(args[1:])))
}

func doDelete(cmd *cobra.Command, args []string) {
	printResponse(api.RawDelete(args[0], readData(cmd), readHeaders(args[1:])))
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
