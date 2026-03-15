//go:build linux

package simulator

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// OpenPTY opens a pseudo-terminal master device and returns the master file
// and the path to the corresponding slave device.
func OpenPTY() (master *os.File, slaveName string, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open ptmx: %w", err)
	}

	slaveName, err = ptsName(master)
	if err != nil {
		_ = master.Close()
		return nil, "", fmt.Errorf("failed to get pts name: %w", err)
	}

	if err := unlockPts(master); err != nil {
		_ = master.Close()
		return nil, "", fmt.Errorf("failed to unlock pts: %w", err)
	}

	return master, slaveName, nil
}

// ptsName returns the name of the pseudo-terminal slave associated with the given master.
func ptsName(f *os.File) (string, error) {
	var n int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return "", errno
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

// unlockPts unlocks the pseudo-terminal slave associated with the given master.
func unlockPts(f *os.File) error {
	var n int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return errno
	}
	return nil
}
