package config

import (
	"path"
	"strings"

	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// TimingConfig holds the timing device settings.
type TimingConfig struct {
	SerialPort    string `mapstructure:"serial_port" json:"serialPort" usage:"Serial port of the timing device, host:port for TCP, empty scans all ports"`
	BaudRate      int    `mapstructure:"baud_rate" json:"baudRate" default:"9600" values:"9600,19200,38400,57600,115200" usage:"Baud rate of the timing device"`
	PoolLength    int    `mapstructure:"pool_length" json:"poolLength" default:"50" usage:"Pool length in meters for the distance per lap, 0 derives it from the event distance and the lap count"`
	SimulateEvent int    `mapstructure:"simulate_event" json:"simulateEvent" default:"0" usage:"Event number the simulator plays instead of reading the timing device, from the replayed recording or with random times, 0 disables it"`
	ReplayPath    string `mapstructure:"replay_path" json:"replayPath" usage:"Storage path of a recording the simulator replays instead of reading the timing device, such as recordings/recording_2026-10-18.jsonl, empty for none"`
	Recording     bool   `mapstructure:"recording" json:"recording" default:"true" usage:"Record the data of the timing device in recordings/ of the local storage"`
}

// validateTiming normalizes and checks the timing device settings.
func validateTiming(cfg *TimingConfig) error {
	var p structconf.Problems

	const (
		// maxPoolLength bounds the pool length in meters
		maxPoolLength = 1000000
		// maxEvent bounds the event number the simulator plays
		maxEvent = 1000000
	)

	cfg.SerialPort = strings.TrimSpace(cfg.SerialPort)
	if len(cfg.SerialPort) > maxText {
		p.Addf("timing.serial_port must be at most %d characters", maxText)
	}

	if cfg.PoolLength < 0 || cfg.PoolLength > maxPoolLength {
		p.Addf("timing.pool_length must be between 0 and %d", maxPoolLength)
	}

	if cfg.SimulateEvent < 0 || cfg.SimulateEvent > maxEvent {
		p.Addf("timing.simulate_event must be between 0 and %d", maxEvent)
	}

	cfg.ReplayPath = strings.TrimSpace(cfg.ReplayPath)
	if cfg.ReplayPath != "" && !strings.EqualFold(path.Ext(cfg.ReplayPath), ".jsonl") {
		p.Addf("timing.replay_path must point to a .jsonl file, got %q", cfg.ReplayPath)
	}
	if len(cfg.ReplayPath) > maxPath {
		p.Addf("timing.replay_path must be at most %d characters", maxPath)
	}

	return p.Err()
}
