package keys

import (
	"github.com/spf13/cobra"
)

var caRenewalCmd = &cobra.Command{
	Use:   "renewal",
	Short: "Renew the root of trust for your factory PKI",
	Long: `These sub-commands allow you to gradually renew a root of trust for your factory PKI.

A guided process allows you to:
- Generate a new root of trust.
- Create an EST standard compliant root CA renewal bundle.- Re-sign all necessary factory PKI certificates.
- Provision a new root of trust to all your devices without service interruption.`,
}

func init() {
	caCmd.AddCommand(caRenewalCmd)
}
