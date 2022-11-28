package targets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/subcommands"
)

var editNoTail bool

func init() {
	editCmd := &cobra.Command{
		Use:    "edit",
		Short:  "Edit targets.json directly - proceed with caution!",
		Run:    doEdit,
		Hidden: true,
	}
	cmd.AddCommand(editCmd)
	editCmd.Flags().BoolVarP(&editNoTail, "no-tail", "", false, "Don't tail output of CI Job")
}

func doEdit(cmd *cobra.Command, args []string) {
	factory := viper.GetString("factory")
	logrus.Debugf("Editing targets for %s", factory)

	// Get raw json
	targets, err := api.TargetsList(factory)
	subcommands.DieNotNil(err)
	orig, err := subcommands.MarshalIndent(targets, "", "  ")
	subcommands.DieNotNil(err)

	// Create temp file to edit with
	tmpfile, err := os.CreateTemp("", "targets.*.json")
	if err != nil {
		fmt.Println("Unable to create tempfile: ", err)
		os.Exit(1)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(orig); err != nil {
		fmt.Println("Unable to write tempfile: ", err)
		os.Exit(1)
	}
	if err := tmpfile.Close(); err != nil {
		fmt.Println("Unable to close tempfile: ", err)
		os.Exit(1)
	}

	// Let user edit the file
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "/usr/bin/vi"
	}
	edit := exec.Command(editor, tmpfile.Name())
	edit.Stdout = os.Stdout
	edit.Stderr = os.Stderr
	edit.Stdin = os.Stdin
	logrus.Debug("Running editor and waiting for it to finish...")
	if err := edit.Run(); err != nil {
		fmt.Println("Editing cancelled: ", err)
		os.Exit(0)
	}

	// Read it and see if its changed
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		fmt.Println("ERROR: Unable to re-read tempfile:", err)
	}
	if bytes.Equal(content, orig) {
		fmt.Println("No changes found, exiting.")
		os.Exit(0)
	}

	// Push changes
	var newTargets tuf.Files
	err = json.Unmarshal(content, &newTargets)
	if err != nil {
		fmt.Println("Unable to parse new targets: ", err)
		os.Exit(1)
	}
	type TargetsUp struct {
		Targets tuf.Files `json:"targets"`
	}
	upload := TargetsUp{newTargets}
	content, err = json.Marshal(upload)
	if err != nil {
		fmt.Println("Unable to marshall targets data: ", err)
		os.Exit(1)
	}

	logrus.Debugf("Pushing to server: %s", string(content))
	jobservUrl, webUrl, err := api.TargetsPut(factory, content)
	subcommands.DieNotNil(err)
	fmt.Printf("CI URL: %s\n", webUrl)
	if !editNoTail {
		api.JobservTail(jobservUrl)
	}
}
