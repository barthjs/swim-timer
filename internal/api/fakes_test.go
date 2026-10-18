package api_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/serial"
	"github.com/barthjs/swim-timer/internal/update"
)

// fakeDevice passes frames of a test to the timing service instead of reading a serial port.
type fakeDevice struct {
	port      string
	connect   bool
	reader    *io.PipeReader
	writer    *io.PipeWriter
	mu        sync.Mutex
	isRunning bool
}

// newFakeDevice returns a fake device on the given port that connects when the application starts.
func newFakeDevice(port string) *fakeDevice {
	reader, writer := io.Pipe()
	return &fakeDevice{
		port:    port,
		connect: true,
		reader:  reader,
		writer:  writer,
	}
}

// Run hands the byte stream to handle until ctx is canceled. A disconnected device only waits.
func (d *fakeDevice) Run(ctx context.Context, handle serial.Handler) {
	if !d.connect {
		<-ctx.Done()
		return
	}

	d.setRunning(true)
	defer d.setRunning(false)

	go func() {
		<-ctx.Done()
		_ = d.writer.Close()
	}()
	_ = handle(ctx, d.reader)
}

// setRunning marks whether the device currently runs.
func (d *fakeDevice) setRunning(running bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.isRunning = running
}

// Port returns the port while the device runs.
func (d *fakeDevice) Port() string {
	if !d.connected() {
		return ""
	}
	return d.port
}

// Ping reports whether the device runs.
func (d *fakeDevice) Ping(context.Context) error {
	if !d.connected() {
		return errors.New("not connected")
	}
	return nil
}

// connected reports whether the device currently runs.
func (d *fakeDevice) connected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.isRunning
}

// Send writes frames as the timing device transmits them.
func (d *fakeDevice) Send(t *testing.T, frames ...omega.FrameData) {
	t.Helper()

	for _, frame := range frames {
		_, err := d.writer.Write(omega.Encode(frame))
		require.NoError(t, err)
	}
}

// newFakeUpdater returns an updater that asks a fake GitHub API, which reports latest as the latest release.
// Without an architecture, it never replaces the binary of the test.
func newFakeUpdater(t *testing.T, latest string) *update.Updater {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.MarshalWrite(w, map[string]any{
			"tag_name": latest,
			"html_url": "https://example.test/releases/" + latest,
			"assets":   []any{},
		})
	}))
	t.Cleanup(srv.Close)

	return update.New(update.Options{
		Repository: "owner/app",
		Binary:     "app",
		Version:    harnessVersion,
		APIURL:     srv.URL,
	})
}
