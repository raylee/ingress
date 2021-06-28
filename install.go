package main

import (
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

//go:embed ingress.service
var serviceDef []byte

const (
	serviceName = "ingress"
	userName    = "svc-" + serviceName
	exeFile     = "/svc/" + serviceName + "/ingress"
	systemdFile = "/etc/systemd/system/" + serviceName + ".service"
	serviceDesc = "api.evq.io ingress manager"
)

func UidGid(name string) (uid, gid int, err error) {
	svcUser, err := user.Lookup(name)
	fmt.Sscan(svcUser.Uid, &uid)
	fmt.Sscan(svcUser.Gid, &gid)
	return uid, gid, err
}

// WriteIfMissing only creates the file if it doesn't exist.
func WriteIfMissing(filename string, data []byte, perm fs.FileMode) {
	_, err := os.Stat(filename)
	if err == nil { // it already exists
		return
	}
	os.WriteFile(filename, data, perm)
}

// Add a system user and group for this process.
func addSystemUser(userName, homeDir, comment string) (uid, gid int, err error) {
	cmd := exec.Command(
		"useradd",
		"--create-home",
		"--home-dir", homeDir,
		"--system",
		"--shell", "/bin/sh",
		"--user-group", // create a group with the same name
		"--comment", comment,
		userName,
	)
	if _, err = cmd.Output(); err != nil {
		return
	}
	if uid, gid, err = UidGid(userName); err != nil {
		err = fmt.Errorf("Could not look up newly-created user %s: %w", userName, err)
		return
	}
	return
}

func shell(cmds ...string) error {
	for _, line := range cmds {
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("Could not execute %s (%s): %w", line, out, err)
		}
	}
	return nil
}

func Install() error {
	var uid, gid int

	if os.Geteuid() != 0 {
		return fmt.Errorf("Install must be done as root")
	}
	StopService()
	os.Mkdir("/svc", 0755)

	_, err := user.Lookup(userName)
	if err != nil {
		uid, gid, err = addSystemUser(userName, "/svc/"+serviceName, serviceDesc)
		if err != nil {
			return fmt.Errorf("Could not add user '%s' to system: %w", userName, err)
		}
	}
	self, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return fmt.Errorf("Could not locate self on disk: %w", err)
	}
	err = os.Chown(self, uid, gid)
	if err != nil {
		return fmt.Errorf("Could not change exe owner to %s: %w", userName, err)
	}
	err = os.Rename(self, exeFile)
	if err != nil {
		return fmt.Errorf("Could not install executable to home dir: %w", err)
	}
	// Always overwrite the systemd service definition to ensure it's up to date.
	err = os.WriteFile(systemdFile, serviceDef, 0644)
	if err != nil {
		return fmt.Errorf("Could not create systemd service file: %w", err)
	}

	// Only write the default /etc/ingress/* files if they don't exist, as these are under user control.
	// WriteIfMissing(configFile, defaultConfig, 0644)
	// WriteIfMissing(secretsFile, defaultSecrets, 0600)

	err = shell(
		"systemctl daemon-reload",
		"systemctl enable "+serviceName,
		"systemctl start "+serviceName,
	)
	if err != nil {
		return fmt.Errorf("Could not execute command: %w", err)
	}

	return nil
}

func Restart() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Restart must be done as root")
	}
	err := shell(
		"systemctl restart " + serviceName,
	)
	return err
}

func StopService() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Stop must be done as root")
	}
	return shell(
		"systemctl stop "+serviceName,
		"systemctl disable "+serviceName,
	)
}

func Uninstall() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Uninstall must be done as root")
	}
	err := shell(
		"systemctl stop "+serviceName,
		"systemctl disable "+serviceName,
		"sleep 1",
		"userdel "+userName,
	)
	return err
}
