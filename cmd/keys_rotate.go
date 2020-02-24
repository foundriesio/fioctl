package cmd

import (
	"fmt"
	"github.com/foundriesio/fioctl/internal/progress"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/foundriesio/fioctl/internal/piper"
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

type rotationExitCode int

const (
	ErrUnknown               rotationExitCode = -1
	RotationSuccess          rotationExitCode = 0
	ErrUnpackCredentials     rotationExitCode = 1
	ErrPullCredentials       rotationExitCode = 2
	ErrParseCredentials      rotationExitCode = 3
	ErrRegenerateCredentials rotationExitCode = 4
	ErrAddNewCredentials     rotationExitCode = 5
	ErrRemoveOldCredentials  rotationExitCode = 6
	ErrSignCredentials       rotationExitCode = 7
	ErrPackCredentials       rotationExitCode = 8
	ErrPushCredentials       rotationExitCode = 9
)

var rotationErrorExitCodes = [9]rotationExitCode{
	ErrUnpackCredentials, ErrPullCredentials, ErrParseCredentials, ErrRegenerateCredentials, ErrAddNewCredentials,
	ErrRemoveOldCredentials, ErrSignCredentials, ErrPackCredentials, ErrPushCredentials}

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
        args, cwd=cwd,stderr=subprocess.STDOUT, stdout=subprocess.PIPE)
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
    sys.exit(3)

with TemporaryDirectory() as tempdir:
    os.chdir(tempdir)
    os.mkdir('tuf')
    creds_file = '/tmp/creds.tgz'
    try:
      cmd('tar', 'xf', creds_file, cwd='./tuf')
    except:
      sys.exit(1)
    try:
      cmd('garage-sign', 'root', 'pull', '--repo', './tufrepo')
    except:
      sys.exit(2)
    try:
      old_keyname, old_keyid = find_current_root('./tuf/tufrepo')
    except:
      sys.exit(3)
    keyname = 'offline-root-' + datetime.datetime.now().isoformat()
    try:
      cmd('garage-sign', 'key', 'generate', '--repo', './tufrepo',
          '--type', 'rsa', '--name', keyname)
    except:
      sys.exit(4)
    try:
      cmd('garage-sign', 'root', 'key', 'add', '--repo', './tufrepo',
          '--key-name', keyname)
    except:
      sys.exit(5)
    try:
      cmd('garage-sign', 'root', 'key', 'remove', '--repo', './tufrepo',
          '--key-id', old_keyid, '--key-name', old_keyname)
    except:
      sys.exit(6)
    try:
      cmd('garage-sign', 'root', 'sign', '--repo', './tufrepo',
          '--key-name', keyname, '--key-name', old_keyname)
    except:
      sys.exit(7)
    try:
      cmd('tar', 'czf', creds_file, 'tufrepo', cwd='./tuf')
    except:
      sys.exit(8)
    try:
      cmd('garage-sign', 'root', 'push', '--repo', './tufrepo')
    except:
      sys.exit(9)
`
	p := progress.New("Pulling aktualizr image %s")
	if err := pullContainer(aktualizrImageName); err != nil {
		p.Fail()
		exitf("failed to pull image, %q, %s", aktualizrImageName, err)
	}
	p.Finish()
	scriptPath, err := loadScript(rotateCmd)
	if err != nil {
		exitf("failed to load script, %s", err)
	}
	defer os.Remove(scriptPath) // clean up
	if err := copyFile(credentialsPath, credentialsBackupPath); err != nil {
		exitf("failed to backup offline credentials, %s", err)
	}
	fmt.Printf("Original credentials --> %q\n", credentialsBackupPath)
	p = progress.New("Rotating offline credentials %s")
	if output, err := runRotationScript(aktualizrImageName, scriptPath, credentialsPath); err != nil {
		p.Fail()
		switch parseErrExitCode(err) {
		case ErrUnpackCredentials:
			exitf("failed to unpack credentials\n%s", output)
		case ErrPullCredentials:
			exitf("failed to pull credentials\n%s", output)
		case ErrParseCredentials:
			exitf("failed to parse credentials\n%s", output)
		case ErrRegenerateCredentials:
			exitf("failed to regenerate credentials\n%s", output)
		case ErrAddNewCredentials:
			exitf("failed to add new credentials\n%s", output)
		case ErrRemoveOldCredentials:
			exitf("failed to remove old credentials\n%s", output)
		case ErrSignCredentials:
			exitf("failed to sign credentials\n%s", output)
		case ErrPackCredentials:
			exitf("failed to pack credentials\n%s", output)
		case ErrPushCredentials:
			exitf("failed to push credentials\n%s", output)
		default:
			exitf("%s\n%s", err, output)
		}
	}
	p.Finish()
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

func parseErrExitCode(err error) rotationExitCode {
	if err == nil {
		return RotationSuccess
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		code := rotationExitCode(exitCode)
		for _, item := range rotationErrorExitCodes {
			if item == code {
				return item
			}
		}
	}
	return ErrUnknown
}

func runRotationScript(imageName string, sourcePath string, credentialsPath string) (string, error) {
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
	return piper.Run(*exec.Command("docker", args...))
}

func pullContainer(name string) error {
	command := fmt.Sprintf("docker pull %s", name)
	if _, err := cli(command); err != nil {
		return err
	}
	return nil
}

func verifyDocker() error {
	if _, err := cli("docker --version"); err != nil {
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

func cli(input string) ([]byte, error) {
	cmd := exec.Command("/bin/sh", "-c", input)
	return cmd.CombinedOutput()
}
