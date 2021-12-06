package el2g

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/foundriesio/fioctl/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	awsCmd := &cobra.Command{
		Use:   "config-aws-iot",
		Short: "Setup EdgeLock2Go support in AWS IOT",
		Run:   doAwsIOT,
	}
	cmd.AddCommand(awsCmd)
}

func doAwsIOT(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")

	fmt.Println("Getting registration code from AWS")
	resp := run("/usr/bin/env", "aws", "iot", "get-registration-code")
	fmt.Println(" |->", resp["registrationCode"])

	fmt.Println("Configuring EdgeLock2Go")
	cert, err := api.El2gConfigAws(factory, resp["registrationCode"])
	subcommands.DieNotNil(err)

	cafile, err := ioutil.TempFile("", "el2g-*.crt")
	subcommands.DieNotNil(err)
	defer os.Remove(cafile.Name())
	_, err = cafile.WriteString(cert.CA)
	subcommands.DieNotNil(err)

	certfile, err := ioutil.TempFile("", "el2g-*.crt")
	subcommands.DieNotNil(err)
	defer os.Remove(certfile.Name())
	_, err = certfile.WriteString(cert.Cert)
	subcommands.DieNotNil(err)

	resp = run("/usr/bin/env", "aws", "iot", "register-ca-certificate",
		"--ca-certificate", "file://"+cafile.Name(),
		"--verification-cert", "file://"+certfile.Name())
	certId := resp["certificateId"]
	fmt.Println(" |-> AWS Certificate ID", certId)

	c := exec.Command("/usr/bin/env", "aws", "iot", "update-ca-certificate",
		"--certificate-id", certId,
		"--new-status", "ACTIVE",
		"--new-auto-registration-status", "ENABLE",
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	subcommands.DieNotNil(c.Run())
	fmt.Println(" |-> ACTIVE")
}

func run(args ...string) map[string]string {
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	subcommands.DieNotNil(cmd.Run())
	var resp map[string]string
	subcommands.DieNotNil(json.Unmarshal(out.Bytes(), &resp))
	return resp
}
