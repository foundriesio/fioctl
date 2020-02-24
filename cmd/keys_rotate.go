package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var keysRotateCmd = &cobra.Command{
	Use:   "rotate <offline key archive>",
	Short: "Rotate root signing key used by the Factory",
	Run:   doKeyRotation,
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	keysCmd.AddCommand(keysRotateCmd)
}

func doKeyRotation(cmd *cobra.Command, args []string) {
	credentialsPath := args[0]
	credentialsBackupPath := fmt.Sprintf("%s.bak", credentialsPath)
	if err := verifyDocker(); err != nil {
		exitf("invalid environment, %s", err)
	}
	const aktualizrImageName = "hub.foundries.io/aktualizr"
	const rotateCmd = `#!/usr/bin/python3
import datetime
import json
import os
import subprocess
import sys
from tempfile import TemporaryDirectory

def cmd(*args, cwd=None):
    print('running: %s' % ' '.join(args))
    p = subprocess.Popen(
        args, cwd=cwd, stderr=subprocess.STDOUT, stdout=subprocess.PIPE)
    for line in p.stdout:
        sys.stdout.buffer.write(line)
    p.wait()
    if p.returncode != 0:
        raise subprocess.CalledProcessError(p.returncode, args)

def find_current_root(repodir):
    print("finding root key name and key id")
    with open(os.path.join(repodir, 'roles/unsigned/root.json')) as f:
        root_role = json.load(f)

    key_ids = root_role['roles']['root']['keyids']
    assert len(key_ids) == 1, "Unexpected number of root keys"
    print("current root keyid:", key_ids[0])
    pubkey = root_role['keys'][key_ids[0]]["keyval"]["public"]

    # now find pubkey:
    keydir = os.path.join(repodir, 'keys')
    for x in os.listdir(keydir):
        if x.endswith('.pub'):
            with open(os.path.join(keydir, x)) as f:
                key = json.load(f)['keyval']['public']
                if pubkey == key:
                    keyname = x[:-4]  # strip off .pub
                    print("current root keyname:", keyname)
                    return keyname, key_ids[0]
    print('could not find root key name')
    sys.exit(1)

with TemporaryDirectory() as tempdir:
    os.chdir(tempdir)
    os.mkdir('tuf')
    creds_file = '/tmp/creds.tgz'
    cmd('tar', 'xf', creds_file, cwd='./tuf')
    cmd('garage-sign', 'root', 'pull', '--repo', './tufrepo')
    old_keyname, old_keyid = find_current_root('./tuf/tufrepo')
    keyname = 'offline-root-' + datetime.datetime.now().isoformat()
    cmd('garage-sign', 'key', 'generate', '--repo', './tufrepo',
    '--type', 'rsa', '--name', keyname)
    cmd('garage-sign', 'root', 'key', 'add', '--repo', './tufrepo',
    '--key-name', keyname)
    cmd('garage-sign', 'root', 'key', 'remove', '--repo', './tufrepo',
    '--key-id', old_keyid, '--key-name', old_keyname)
    cmd('garage-sign', 'root', 'sign', '--repo', './tufrepo',
    '--key-name', keyname, '--key-name', old_keyname)
    cmd('tar', 'czf', creds_file, 'tufrepo', cwd='./tuf')
    cmd('garage-sign', 'root', 'push', '--repo', './tufrepo')
`
	fmt.Println("Pulling aktualizr image...")
	if err := pullContainer(aktualizrImageName); err != nil {
		exitf("failed to pull image, %q, %s", aktualizrImageName, err)
	}
	scriptPath, err := loadScript(rotateCmd)
	if err != nil {
		exitf("failed to load script, %s", err)
	}
	defer os.Remove(scriptPath) // clean up
	if err := copyFile(credentialsPath, credentialsBackupPath); err != nil {
		exitf("failed to backup offline credentials, %s", err)
	}
	fmt.Printf("Original credentials --> %q\n", credentialsBackupPath)
	fmt.Println("Rotating offline credentials...")
	if err := runRotationScript(aktualizrImageName, scriptPath, credentialsPath); err != nil {
		exitf("failed key rotation %s", err)
	}
	fmt.Printf("Updated credentials --> %q\n", credentialsPath)
}

func exitf(format string, args ...interface{}) {
	errFormat := fmt.Sprintf("Error: %s\n", format)
	fmt.Printf(errFormat, args...)
	os.Exit(1)
}

func loadScript(script string) (string, error) {
	content := []byte(script)
	tmpFile, err := ioutil.TempFile("/tmp", "*tmpScript")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary script: %w", err)
	}
	scriptPath := tmpFile.Name()
	if _, err := tmpFile.Write(content); err != nil {
		return "", fmt.Errorf("failed to write temporary script: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to save temporary script: %w", err)
	}
	if err := os.Chmod(scriptPath, 0777); err != nil {
		return "", fmt.Errorf("failed to make script executable: %w", err)
	}
	return scriptPath, nil
}

func runRotationScript(imageName string, sourcePath string, credentialsPath string) error {
	targetPath := "/tmp/tmp.py"
	args := []string{
		// base args
		"run", "--rm",
		// env args
		"--env", "PYTHONUNBUFFERED=1",
		// mount args
		"-v", fmt.Sprintf("%s:/tmp/creds.tgz", credentialsPath),
		"-v", fmt.Sprintf("%s:%s", sourcePath, targetPath),
		// load args
		imageName, targetPath,
	}
	return RunStreamed("docker", args...)
}

func pullContainer(name string) error {
	command := fmt.Sprintf("docker pull %s", name)
	return cli(command)
}

func verifyDocker() error {
	if err := cli("docker --version"); err != nil {
		return fmt.Errorf("docker not available")
	}
	return nil
}

func copyFile(source string, target string) error {
	from, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening credentials path failed: %w", err)
	}
	defer from.Close()
	fromStat, err := from.Stat()
	if err != nil {
		return fmt.Errorf("credentials details retrieval failed: %w", err)
	}
	to, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("opening backup path failed: %w", err)
	}
	defer to.Close()
	if _, err = io.Copy(to, from); err != nil {
		return err
	}
	if err = to.Chmod(fromStat.Mode()); err != nil {
		return fmt.Errorf("backup permissions failed: %w", err)
	}
	return nil
}

func cli(input string) error {
	return RunStreamed("/bin/sh", "-c", input)
}

//Allows tests to mock this command
var execCommand = exec.Command

func errorIndent(content string) string {
	return "| " + strings.Replace(content, "\n", "\n| ", -1) + "_"
}

func RunFrom(fromDir string, command string, args ...string) (string, error) {
	cmd := execCommand(command, args...)
	cmd.Dir = fromDir
	binaryOut, err := cmd.CombinedOutput()
	out := string(binaryOut)
	if err != nil {
		return "", fmt.Errorf("Unable to run '%s'. err(%s), output=\n%s",
			cmd.Args, err, errorIndent(out))
	}

	return out, nil
}

func Run(command string, args ...string) (string, error) {
	return RunFrom("", command, args...)
}

func RunFromStreamedTo(fromDir string, stdOut, stdErr io.Writer, command string, args ...string) error {
	cmd := execCommand(command, args...)
	cmd.Dir = fromDir
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Unable to run '%s': err=%s", cmd.Args, err)
	}
	return nil
}

func RunStreamed(command string, args ...string) error {
	return RunFromStreamedTo("", os.Stdout, os.Stderr, command, args...)
}
