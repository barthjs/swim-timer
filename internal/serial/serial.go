// Package serial provides byte-stream access to serial devices behind a driver-agnostic interface,
// with interchangeable local serial port and TCP (serial-over-IP) backends.
package serial

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// driver selects how a port is opened.
type driver string

const (
	driverDevice driver = "device"
	driverTCP    driver = "tcp"
)

// Parity is the parity setting of a serial port.
type Parity string

const (
	ParityNone  Parity = "none"
	ParityOdd   Parity = "odd"
	ParityEven  Parity = "even"
	ParityMark  Parity = "mark"
	ParitySpace Parity = "space"
)

// StopBits is the number of stop bits of a serial port.
type StopBits string

const (
	StopBitsOne          StopBits = "1"
	StopBitsOnePointFive StopBits = "1.5"
	StopBitsTwo          StopBits = "2"
)

// tcpDialTimeout bounds the connection to a serial-over-IP bridge, so an unreachable bridge is retried soon
// and a shutdown does not wait for the timeout of the operating system.
const tcpDialTimeout = 5 * time.Second

var (
	// ErrUnsupported is returned when a feature is not available on the current platform.
	ErrUnsupported = errors.New("serial: not supported on this platform")
)

// Mode holds the line settings of a local serial port.
// They are ignored by the TCP driver, where the remote end owns the line settings.
type Mode struct {
	BaudRate int
	DataBits int
	Parity   Parity
	StopBits StopBits
}

// open opens the port with the given name for reading. Canceling ctx aborts a pending connection.
// The caller must close the returned reader.
func open(ctx context.Context, name string, mode Mode) (io.ReadCloser, error) {
	switch driverFor(name) {
	case driverTCP:
		return openTCP(ctx, name)
	case driverDevice:
		return openDevice(name, mode)
	default:
		return nil, fmt.Errorf("serial: unknown driver for port %q", name)
	}
}

// driverFor selects the driver for a port name. A name in host:port form addresses a serial-over-IP bridge,
// such as ser2net or a Moxa device server, which is also how OpenVirtual exposes its port on Windows.
// Everything else is a local device. A name with a path separator is always a device,
// since persistent Linux device paths such as /dev/serial/by-path/pci-0000:00:14.0-usb-0:2:1.0-port0 contain colons as well.
func driverFor(name string) driver {
	if strings.ContainsAny(name, `/\`) {
		return driverDevice
	}

	host, port, err := net.SplitHostPort(name)
	if err != nil || host == "" {
		return driverDevice
	}

	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return driverDevice
	}
	return driverTCP
}

// listPorts returns the names of all serial ports found on the system.
func listPorts() ([]string, error) {
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("serial: listing ports: %w", err)
	}

	names := make([]string, 0, len(details))
	for _, port := range details {
		names = append(names, port.Name)
	}
	return names, nil
}

// openDevice opens a local serial port with the given line settings.
func openDevice(name string, mode Mode) (io.ReadCloser, error) {
	parity, err := deviceParity(mode.Parity)
	if err != nil {
		return nil, err
	}

	stopBits, err := deviceStopBits(mode.StopBits)
	if err != nil {
		return nil, err
	}

	return serial.Open(name, &serial.Mode{
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	})
}

// deviceParity maps a Parity to the parity setting of the serial library.
func deviceParity(parity Parity) (serial.Parity, error) {
	switch parity {
	case ParityNone, "":
		return serial.NoParity, nil
	case ParityOdd:
		return serial.OddParity, nil
	case ParityEven:
		return serial.EvenParity, nil
	case ParityMark:
		return serial.MarkParity, nil
	case ParitySpace:
		return serial.SpaceParity, nil
	default:
		return 0, fmt.Errorf("serial: unknown parity %q", parity)
	}
}

// deviceStopBits maps StopBits to the stop bits setting of the serial library.
func deviceStopBits(stopBits StopBits) (serial.StopBits, error) {
	switch stopBits {
	case StopBitsOne, "":
		return serial.OneStopBit, nil
	case StopBitsOnePointFive:
		return serial.OnePointFiveStopBits, nil
	case StopBitsTwo:
		return serial.TwoStopBits, nil
	default:
		return 0, fmt.Errorf("serial: unknown stop bits %q", stopBits)
	}
}

// openTCP connects to a serial-over-IP bridge at the given host:port address within tcpDialTimeout.
func openTCP(ctx context.Context, address string) (io.ReadCloser, error) {
	dialer := net.Dialer{
		Timeout: tcpDialTimeout,
	}
	return dialer.DialContext(ctx, "tcp", address)
}
