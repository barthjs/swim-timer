package handlers

import (
	"net/http"

	"github.com/barthjs/swim-timer/internal/config"
)

type ConfigInput struct {
	ConfigFilePath string `json:"config_file_path"`
	LogLevel       string `json:"log_level" validate:"required,oneof=debug info warn error"`
	PoolLength     int    `json:"pool_length" validate:"required,min=1,max=1000000"`
	PresetDir      string `json:"preset_dir" validate:"required,min=1,max=255"`
	RaceDataDir    string `json:"race_data_dir" validate:"required,min=1,max=255"`
	SerialPort     string `json:"serial_port" validate:"max=255"`
	SimulateEvent  int    `json:"simulate_event" validate:"min=0,max=1000000"`
	Server         struct {
		Host string `json:"host" validate:"max=255"`
		Port int    `json:"port" validate:"required,min=1,max=65535"`
	} `json:"server" validate:"required"`
	Secret string `json:"secret" validate:"max=512"`
}

// showConfigHandler returns the current loaded configuration.
func (h *Handlers) showConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, h.cfg, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// updateConfigHandler updates the config in the loaded config file.
func (h *Handlers) updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var input ConfigInput

	if err := readJSON(w, r, &input); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	if errs := validate(input); errs != nil {
		h.failedValidationResponse(w, r, errs)
		return
	}

	cfg := config.Config{
		ConfigFilePath: h.cfg.ConfigFilePath,
		RawLogLevel:    input.LogLevel,
		PoolLength:     input.PoolLength,
		PresetsDir:     input.PresetDir,
		RaceDataDir:    input.RaceDataDir,
		SerialPort:     input.SerialPort,
		SimulateEvent:  input.SimulateEvent,
		Secret:         input.Secret,
		Server: config.Server{
			Host: input.Server.Host,
			Port: input.Server.Port,
		},
	}

	if err := config.WriteConfig(cfg); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, h.cfg, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
