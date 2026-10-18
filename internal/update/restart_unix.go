//go:build !windows

package update

import (
	"os"
	"syscall"
)

// Restart replaces the running process with the binary Apply has updated.
// The process keeps its ID, so a supervisor such as systemd does not notice the restart,
// and supervised has no effect.
func (u *Updater) Restart(supervised bool) error {
	executable, err := u.restartPath()
	if err != nil {
		return err
	}
	return syscall.Exec(executable, os.Args, os.Environ())
}
