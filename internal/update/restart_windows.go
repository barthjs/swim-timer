//go:build windows

package update

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows/svc"
)

// Restart starts the binary Apply has updated with the arguments of the running process.
// The caller exits afterward. A Windows service, or a process a supervisor restarts once it exits,
// is restarted from outside instead, so Restart starts nothing there.
func (u *Updater) Restart(supervised bool) error {
	if supervised {
		return nil
	}
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if isService {
		return nil
	}

	executable, err := u.restartPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
