//go:build !linux && !windows

package serial

import "io"

// OpenVirtual is not available on this platform.
func OpenVirtual() (io.WriteCloser, string, error) {
	return nil, "", ErrUnsupported
}
