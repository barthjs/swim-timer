package serial

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("opens a TCP port", func(t *testing.T) {
		t.Parallel()

		address := listen(t, []byte("frame"))

		conn, err := open(t.Context(), address, Mode{})

		require.NoError(t, err)
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "frame", string(buf))
	})

	t.Run("rejects a device that does not exist", func(t *testing.T) {
		t.Parallel()

		_, err := open(t.Context(), "/dev/does-not-exist", Mode{
			BaudRate: 9600,
			DataBits: 8,
		})

		assert.Error(t, err)
	})

	t.Run("stops when canceled", func(t *testing.T) {
		t.Parallel()

		address := listen(t, []byte("frame"))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := open(ctx, address, Mode{})

		assert.ErrorIs(t, err, context.Canceled)
	})
}

// listen starts a TCP server on a free local port that sends payload to every client
// and keeps the connection open until the test ends.
func listen(t *testing.T, payload []byte) string {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write(payload)
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	return listener.Addr().String()
}

func TestDriverFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		port string
		want driver
	}{
		{
			name: "selects the device driver for a linux device",
			port: "/dev/ttyUSB0",
			want: driverDevice,
		},
		{
			name: "selects the device driver for a windows device",
			port: "COM3",
			want: driverDevice,
		},
		{
			name: "selects the TCP driver for a host and port",
			port: "192.168.1.20:4001",
			want: driverTCP,
		},
		{
			name: "selects the TCP driver for a hostname and port",
			port: "bridge.local:950",
			want: driverTCP,
		},
		{
			name: "selects the TCP driver for an ipv6 address and port",
			port: "[::1]:4001",
			want: driverTCP,
		},
		{
			name: "selects the device driver for a linux device path with colons",
			port: "/dev/serial/by-path/pci-0000:00:14.0-usb-0:2:1.0-port0",
			want: driverDevice,
		},
		{
			name: "selects the device driver for a windows device path",
			port: `\\.\COM12`,
			want: driverDevice,
		},
		{
			name: "selects the device driver without a host",
			port: ":4001",
			want: driverDevice,
		},
		{
			name: "selects the device driver for a non-numeric port",
			port: "bridge.local:serial",
			want: driverDevice,
		},
		{
			name: "selects the device driver for a port out of range",
			port: "bridge.local:70000",
			want: driverDevice,
		},
		{
			name: "selects the device driver for port zero",
			port: "bridge.local:0",
			want: driverDevice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := driverFor(tt.port)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenDevice(t *testing.T) {
	t.Parallel()

	t.Run("rejects an invalid parity", func(t *testing.T) {
		t.Parallel()

		_, err := openDevice("/dev/null", Mode{
			Parity: "invalid",
		})

		assert.Error(t, err, "invalid parity")
	})

	t.Run("rejects invalid stop bits", func(t *testing.T) {
		t.Parallel()

		_, err := openDevice("/dev/null", Mode{
			StopBits: "3",
		})

		assert.Error(t, err, "invalid stop bits")
	})
}

func TestDeviceParity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Parity
		want serial.Parity
	}{
		{
			name: "maps an empty parity to none",
			in:   "",
			want: serial.NoParity,
		},
		{
			name: "maps none",
			in:   ParityNone,
			want: serial.NoParity,
		},
		{
			name: "maps odd",
			in:   ParityOdd,
			want: serial.OddParity,
		},
		{
			name: "maps even",
			in:   ParityEven,
			want: serial.EvenParity,
		},
		{
			name: "maps mark",
			in:   ParityMark,
			want: serial.MarkParity,
		},
		{
			name: "maps space",
			in:   ParitySpace,
			want: serial.SpaceParity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := deviceParity(tt.in)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects an unknown parity", func(t *testing.T) {
		t.Parallel()

		_, err := deviceParity("invalid")

		assert.Error(t, err)
	})
}

func TestDeviceStopBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   StopBits
		want serial.StopBits
	}{
		{
			name: "maps empty stop bits to one",
			in:   "",
			want: serial.OneStopBit,
		},
		{
			name: "maps one",
			in:   StopBitsOne,
			want: serial.OneStopBit,
		},
		{
			name: "maps one and a half",
			in:   StopBitsOnePointFive,
			want: serial.OnePointFiveStopBits,
		},
		{
			name: "maps two",
			in:   StopBitsTwo,
			want: serial.TwoStopBits,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := deviceStopBits(tt.in)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects unknown stop bits", func(t *testing.T) {
		t.Parallel()

		_, err := deviceStopBits("3")

		assert.Error(t, err)
	})
}
