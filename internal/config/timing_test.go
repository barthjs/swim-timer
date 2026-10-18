package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTiming(t *testing.T) {
	t.Parallel()

	t.Run("trims the serial port", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Timing.SerialPort = " COM3 "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, "COM3", cfg.Timing.SerialPort)
	})

	t.Run("trims the replay path", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Timing.ReplayPath = " recordings/recording.JSONL "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, "recordings/recording.JSONL", cfg.Timing.ReplayPath)
	})

	t.Run("accepts a replay path together with a simulated event", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Timing.SimulateEvent = 1
		cfg.Timing.ReplayPath = "recordings/recording.jsonl"

		err := Validate(&cfg)

		assert.NoError(t, err, "the simulator replays the event from the recording")
	})

	t.Run("accepts every baud rate", func(t *testing.T) {
		t.Parallel()

		for _, baudRate := range []int{9600, 19200, 38400, 57600, 115200} {
			cfg := testConfig(t)
			cfg.Timing.BaudRate = baudRate

			err := Validate(&cfg)

			assert.NoError(t, err, baudRate)
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "negative pool length",
				change:  func(c *Config) { c.Timing.PoolLength = -1 },
				wantErr: "timing.pool_length",
			},
			{
				name:    "negative event",
				change:  func(c *Config) { c.Timing.SimulateEvent = -1 },
				wantErr: "timing.simulate_event",
			},
			{
				name:    "replay path without a recording",
				change:  func(c *Config) { c.Timing.ReplayPath = "recordings/recording.csv" },
				wantErr: "timing.replay_path",
			},
			{
				name:    "long replay path",
				change:  func(c *Config) { c.Timing.ReplayPath = strings.Repeat("a", 1025) + ".jsonl" },
				wantErr: "timing.replay_path",
			},
			{
				name:    "unsupported baud rate",
				change:  func(c *Config) { c.Timing.BaudRate = 4800 },
				wantErr: "timing.baud_rate",
			},
			{
				name:    "long serial port",
				change:  func(c *Config) { c.Timing.SerialPort = strings.Repeat("a", 256) },
				wantErr: "timing.serial_port",
			},
		})
	})
}
