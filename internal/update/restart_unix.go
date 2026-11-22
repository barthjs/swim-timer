//go:build !windows

package update

import (
	"os"
	"syscall"
)

// Restart replaces the running process with the binary Apply has updated, keeping its ID,
// so a supervisor does not notice and supervised has no effect.
func (u *Updater) Restart(supervised bool) error {
	executable, err := u.restartPath()
	if err != nil {
		return err
	}
	return syscall.Exec(executable, os.Args, os.Environ())
}
