package keys

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the EST TLS certificate authorized for this factory",
		Run:   doShowEst,
	}
	estCmd.AddCommand(cmd)
	cmd.Flags().BoolVarP(&prettyFormat, "pretty", "", false, "Display human readable output of each certificate")

	cmd = &cobra.Command{
		Use:   "authorize <PKI directory>",
		Short: "Authorize Foundries.io to run an EST server at <repoid>.est.foundries.io",
		Run:   doAuthorizeEst,
		Args:  cobra.ExactArgs(1),
		Long: `This command will initiate a transaction with api.foundries.io that:

  * api.foundries.io will create a new private key and TLS certificate signing request
  * This command will sign the request using the Factory's root key.
  * Upload the resultant TLS certificate to api.foundries.io`,
	}
	estCmd.AddCommand(cmd)
}

func doShowEst(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Showing EST cert for %s", factory)

	cert, err := api.FactoryGetCA(factory)
	subcommands.DieNotNil(err)
	if len(cert.EstCrt) == 0 {
		fmt.Println("EST TLS certificate has not been configured for this factory.")
	} else if prettyFormat {
		prettyPrint(cert.EstCrt)
	} else {
		fmt.Println(cert.EstCrt)
	}
}

func doAuthorizeEst(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Authorizing EST for %s", factory)
	pkiDir := args[0]

	signPath := filepath.Join(pkiDir, "sign_tls_csr")
	_, err := os.Stat(signPath)
	subcommands.DieNotNil(err)

	tmpDir, err := os.MkdirTemp("", "*-fio-est")
	subcommands.DieNotNil(err)
	defer os.RemoveAll(tmpDir)

	// Workaround: the sign_tls_csr script has a bug where it expects a
	// file named factory_ca.srl to exist in the working directory.
	subcommands.DieNotNil(
		os.WriteFile(filepath.Join(tmpDir, "factory_ca.srl"), []byte{}, 0o700))

	tmpFile, err := os.Create(filepath.Join(tmpDir, "fio-est.csr"))
	subcommands.DieNotNil(err)

	csr, err := api.FactoryCreateEstCsr(factory)
	subcommands.DieNotNil(err)

	_, err = tmpFile.WriteString(csr)
	subcommands.DieNotNil(err)

	c := exec.Command(signPath, tmpFile.Name())
	buf := bytes.NewBuffer([]byte{})
	stdErr := bytes.NewBuffer([]byte{})
	c.Stdout = buf
	c.Stderr = stdErr
	c.Dir = tmpDir
	subcommands.DieNotNil(c.Run(), stdErr.String())
	cert := buf.String()
	fmt.Println("Uploading new EST certificate:")
	fmt.Println(cert)
	subcommands.DieNotNil(api.FactorySetEstCrt(factory, cert))
}
