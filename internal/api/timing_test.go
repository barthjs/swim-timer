package api_test

import (
	"net/http"
	"testing"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
)

func TestTimingAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/timing/status",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestTimingStatusGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the status of the connected device", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/timing/status")

		response.AssertOK().AssertJSON(map[string]any{
			"connected":      true,
			"port":           h.Device.port,
			"configuredPort": "",
			"simulateEvent":  0,
			"replayFile":     "",
		})
	})

	t.Run("returns the configuration of a disconnected device", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		port := "/dev/tty" + f.LetterN(4)
		event := f.IntRange(1, 999)
		h := newHarness(t, withDisconnectedDevice(), withConfig(func(cfg *config.Config) {
			cfg.Timing.SerialPort = port
			cfg.Timing.SimulateEvent = event
		}))

		response := h.Client.GET("/api/timing/status")

		response.AssertOK().AssertJSON(map[string]any{
			"connected":      false,
			"port":           "",
			"configuredPort": port,
			"simulateEvent":  event,
			"replayFile":     "",
		})
	})

	t.Run("returns the name of the replayed recording", func(t *testing.T) {
		t.Parallel()
		name := "recording_" + apitest.Faker(t).Date().Format("2006-01-02") + ".jsonl"
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Timing.ReplayPath = "recordings/" + name
		}))

		response := h.Client.GET("/api/timing/status")

		response.AssertOK().AssertJSON(map[string]any{
			"connected":      true,
			"port":           h.Device.port,
			"configuredPort": "",
			"simulateEvent":  0,
			"replayFile":     name,
		})
	})
}
