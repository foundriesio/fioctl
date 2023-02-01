package keys

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/foundriesio/fioctl/subcommands"
)

// This allows to chain several TUF updates subcommands into a single "shortcut" worklfow.
var isTufUpdatesShortcut, isTufUpdatesInitialized bool

var tufUpdatesCmd = &cobra.Command{
	Use:   "updates",
	Short: "Manage updates to the TUF root for your factory (expert mode)",
	Long: `These sub-commands allow you to transactionally stage and apply changes
to your Factory's TUF private keys in a granular way familiar for TUF experts.

The TUF updates transaction starts by running the "fioctl keys tuf updates init" command.
That command returns a unique secure Transaction ID which is then required for other actions.
The admin initiating the transaction should save that TXID for the timespan of the transaction.
It must only be shared with those Factory admins which will participate in the transaction.

Typically, admin(s) will run other subcommands to make changes to the TUF root (see examples).
The staged changes can be checked using the "fioctl keys tuf updates review" command.

Finally, the transaction can be applied using the "fioctl keys tuf updates apply" command.
If admin decides to abandon the staged changes they can run "fioctl keys tuf updates cancel".

For increased safety there can be only one active TUF updates transaction at a time.`,
	Example: `
- Take ownership of TUF root and targets keys for a new factory, keep them on separate machines:
  1. On TUF root admin's shell:
     fioctl keys tuf updates init --first-time --keys=tuf-root-keys.tgz
  2. The above command prints a transaction ID (e.g. abcdef42) to be shared with TUF targets admin.
  3. On TUF targets admin's shell:
     fioctl keys tuf updates rotate-offline-key \
	    --role=targets --txid=abcdef42 --targets-keys=tuf-targets-keys.tgz
  4. On TUF root admin's shell:
     fioctl keys tuf updates rotate-offline-key \
	    --role=root --txid=abcdef42 --keys=tuf-root-keys.tgz --sign
  5. On TUF root admin's shell:
     fioctl keys tuf updates apply --txid=abcdef42`,
}

func init() {
	tufCmd.AddCommand(tufUpdatesCmd)

	subcommands.AddLastWill(func() {
		if isTufUpdatesShortcut {
			if isTufUpdatesInitialized {
				// Tuf updates initialized; but the shortcut failed before trying to apply it.
				fmt.Println(`
No changes were made to your Factory.
Please, cancel the staged TUF root updates
using the "fioctl keys tuf updates cancel" command, and try again later.`)
			} else {
				// The init phase failed itself, so there is no active transaction.
				fmt.Println(`
No changes were made to your Factory.
Please, fix an error above and try again.`)
			}
		}
	})
}
