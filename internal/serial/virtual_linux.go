package serial

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

// OpenVirtual creates a virtual serial port backed by a pseudo-terminal.
// Bytes written to the returned writer can be read by opening the returned port name,
// which goes through the same serial device path as a real port.
func OpenVirtual() (io.WriteCloser, string, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, "", fmt.Errorf("serial: opening ptmx: %w", err)
	}

	name, err := ptsName(master)
	if err != nil {
		_ = master.Close()
		return nil, "", fmt.Errorf("serial: getting pts name: %w", err)
	}

	if err := unlockPts(master); err != nil {
		_ = master.Close()
		return nil, "", fmt.Errorf("serial: unlocking pts: %w", err)
	}

	return master, name, nil
}

// ptsName returns the name of the pseudo-terminal slave associated with the given master.
func ptsName(f *os.File) (string, error) {
	// The kernel writes an unsigned int, which a Go int would only partly hold on a big-endian system
	var n uint32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return "", errno
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

// unlockPts unlocks the pseudo-terminal slave associated with the given master.
func unlockPts(f *os.File) error {
	var n int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return errno
	}
	return nil
}
