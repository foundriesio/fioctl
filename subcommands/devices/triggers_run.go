package devices

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

func init() {
	cmd := &cobra.Command{
		Use:   "run <device> <action>",
		Short: "Trigger remote actions on device",
		Long: `*NOTE*: Requires devices running LmP version 97 or later.

This command sets a fioconfig entry that will let a device know it needs to
run a given trigger *once* and report its results back to the device gateway's
"tests" API. You can take the "command ID" from this command's output to then
look up the results using the "fioctl devices tests" command.

# Initiate a remote trigger:
$ fioctl devices triggers run <device> <trigger>
`,
		Args: cobra.ExactArgs(2),
		Run:  doRunTrigger,
	}
	cmd.Flags().StringP("reason", "r", "", "The reason for running this command")
	triggersCmd.AddCommand(cmd)
}

func doRunTrigger(cmd *cobra.Command, args []string) {
	name := args[0]

	// Quick sanity check for device
	d := getDevice(cmd, name)

	// See what triggers are allowed
	allowed := loadRemoteActions(d.Api)

	action := args[1]
	if !slices.Contains(allowed, action) {
		err := fmt.Errorf("Invalid action: %s. Allowed actions are: %s", action, allowed)
		subcommands.DieNotNil(err)
	}
	reason, _ := cmd.Flags().GetString("reason")

	// For the random part anything >= 15 characters will give us about 1,000
	// years before a collision (assuming 100 IDs per second):
	idLen := 15
	maxActionLen := 48 - idLen - 1  // 1 for the underscore in the CommandId below
	if len(action) > maxActionLen { // Device Gateway has max len of 48
		subcommands.DieNotNil(fmt.Errorf("Action name(%s) too long. Max length is %d", action, maxActionLen))
	}
	id := rand.Text()[:idLen]

	opts := triggerOptions{
		Capture:   true,
		Command:   action,
		CommandId: fmt.Sprintf("%s_%s", action, id),
		Reason:    reason,
	}

	ccr := opts.AsConfig()
	subcommands.DieNotNil(d.Api.PatchConfig(ccr, false))
	fmt.Println("Config change submitted. Command ID is:", opts.CommandId)
	fmt.Printf("Use 'fioctl devices tests %s %s' to check results.\n", name, opts.CommandId)
}

type triggerOptions struct {
	CommandId string
	Command   string
	Capture   bool
	Reason    string
}

func (o triggerOptions) AsConfig() client.ConfigCreateRequest {
	b := new(bytes.Buffer)
	capture := 0
	if o.Capture {
		capture = 1
	}
	fmt.Fprintf(b, "COMMAND_ID=%s\n", o.CommandId)
	fmt.Fprintf(b, "COMMAND_CAPTURE=%d\n", capture)

	return client.ConfigCreateRequest{
		Reason: o.Reason,
		Files: []client.ConfigFile{
			{
				Name:        "fioconfig-oneshot-" + o.Command,
				Value:       b.String(),
				Unencrypted: true,
				OnChanged:   []string{"/usr/share/fioconfig/handlers/fioconfig-oneshot", o.Command},
			},
		},
	}
}
