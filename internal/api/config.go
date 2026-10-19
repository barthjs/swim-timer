package api

import (
	"net/http"
	"slices"

	"github.com/go-playground/validator/v10"

	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/service"
)

// configGetHandler returns the effective application configuration without its secrets.
//
//	@Summary		Get config
//	@Description	Returns the effective application configuration with the default and the source of each setting. Secrets are returned empty (Admin only).
//	@Tags			config
//	@Produce		json
//	@Success		200	{object}	ConfigResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/config [get]
func (a *API) configGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeConfig(w, r)
}

// ConfigResponse is the effective configuration without its secrets, with the default and the source of every setting.
// Secrets reports which secrets are set, since their values never leave the backend.
type ConfigResponse struct {
	ConfigSettings
	Secrets  map[string]bool         `json:"secrets"`
	Defaults ConfigSettings          `json:"defaults"`
	Sources  map[string]ConfigSource `json:"sources"`
	File     ConfigFile              `json:"file"`
}

// ConfigSettings are the sections of the configuration.
type ConfigSettings struct {
	App     AppSettings     `json:"app"`
	Log     LogSettings     `json:"log"`
	Server  ServerSettings  `json:"server"`
	Storage StorageSettings `json:"storage"`
	Theme   ThemeSettings   `json:"theme"`
	Timing  TimingSettings  `json:"timing"`
	Webhook WebhookSettings `json:"webhook"`
}

// ConfigSource is where the value of a setting comes from. A flag or an environment variable
// overrides the config file, so a change saved to the file has no effect on such a setting.
type ConfigSource struct {
	Kind string `json:"kind" enums:"default,file,env,flag"`
	// Name is the flag, the environment variable, or the file path, empty for the default.
	Name string `json:"name"`
}

// ConfigFile is the config file a save writes to. Path is empty if none was found at startup,
// a save then creates one. Exists is false until the file at Path is created.
type ConfigFile struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// AppSettings is the app section of the configuration.
type AppSettings struct {
	Name     string `json:"name" validate:"max=100"`
	Env      string `json:"env" enums:"development,staging,production"`
	Language string `json:"language" validate:"oneof=en de" enums:"en,de"`
	Secret   string `json:"secret"`
}

// LogSettings is the log section of the configuration.
type LogSettings struct {
	Level      string `json:"level"`
	Channel    string `json:"channel" validate:"oneof=stdout file both" enums:"stdout,file,both"`
	Path       string `json:"path" validate:"max=1024"`
	Format     string `json:"format" validate:"oneof=json console" enums:"json,console"`
	MaxSizeMB  int    `json:"maxSizeMb" validate:"min=0"`
	MaxAgeDays int    `json:"maxAgeDays" validate:"min=-1"`
	Compress   bool   `json:"compress"`
}

// ServerSettings checks the trusted proxies and the CORS origins as the config package reads them,
// so a value it would reject is a 422 at its field.
type ServerSettings struct {
	Host                 string   `json:"host"`
	Port                 int      `json:"port" validate:"min=1,max=65535"`
	TrustedProxies       string   `json:"trustedProxies" validate:"max=1024,networks"`
	CORSAllowedOrigins   []string `json:"corsAllowedOrigins" validate:"max=50,dive,max=255,origin"`
	CORSAllowCredentials bool     `json:"corsAllowCredentials"`
	SSLEnabled           bool     `json:"sslEnabled"`
	SSLCertFile          string   `json:"sslCertFile" validate:"required_with=SSLKeyFile,max=1024"`
	SSLKeyFile           string   `json:"sslKeyFile" validate:"required_with=SSLCertFile,max=1024"`
}

// validateNetworks reports whether a text lists networks as the config package reads them. It is the "networks" tag,
// so a list the config package would reject is a 422 at its field instead of a 400 for the whole configuration.
func validateNetworks(fl validator.FieldLevel) bool {
	_, err := config.ParseNetworks(fl.Field().String())
	return err == nil
}

// validateOrigin reports whether a text lists allowed CORS origins as the config package reads them.
// It is the "origin" tag.
func validateOrigin(fl validator.FieldLevel) bool {
	_, err := config.ParseOriginList(fl.Field().String())
	return err == nil
}

// validateServerSettings rejects cookies for every origin, since a browser refuses credentials for the wildcard.
// The rule spans two fields, so a struct-level validation reports it at the switch of the credentials.
func validateServerSettings(sl validator.StructLevel) {
	req := sl.Current().Interface().(ServerSettings) //nolint:errcheck // The validation is registered for this type only.
	if !req.CORSAllowCredentials {
		return
	}
	for _, value := range req.CORSAllowedOrigins {
		// An invalid entry is reported by the origin tag, and the wildcard is checked as the config package reads it
		origins, _ := config.ParseOriginList(value)
		if slices.Contains(origins, "*") {
			sl.ReportError(req.CORSAllowCredentials, "corsAllowCredentials", "corsAllowCredentials", "explicit_origins", "")
			return
		}
	}
}

// StorageSettings is the storage section of the configuration.
type StorageSettings struct {
	Driver string     `json:"driver" validate:"oneof=local s3" enums:"local,s3"`
	Path   string     `json:"path" validate:"max=1024"`
	S3     S3Settings `json:"s3"`
}

// S3Settings are the settings of the S3 storage.
type S3Settings struct {
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	Bucket       string `json:"bucket"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	UsePathStyle bool   `json:"usePathStyle"`
}

// validateStorageSettings requires the S3 fields media.New needs once the s3 driver is selected,
// mirroring the storage section of the config package. The secret key may be empty,
// since the admin interface never receives it, the config service checks it after keeping the current one.
func validateStorageSettings(sl validator.StructLevel) {
	req := sl.Current().Interface().(StorageSettings) //nolint:errcheck // The validation is registered for this type only.
	if req.Driver != string(media.DriverS3) {
		return
	}

	if req.S3.Endpoint == "" {
		sl.ReportError(req.S3.Endpoint, "s3.endpoint", "s3.endpoint", "required_if", "")
	}
	if req.S3.Bucket == "" {
		sl.ReportError(req.S3.Bucket, "s3.bucket", "s3.bucket", "required_if", "")
	}
	if req.S3.AccessKey == "" {
		sl.ReportError(req.S3.AccessKey, "s3.accessKey", "s3.accessKey", "required_if", "")
	}
}

// ThemeSettings is the theme section of the configuration.
type ThemeSettings struct {
	Primary   string `json:"primary" validate:"omitempty,hexcolor"`
	Secondary string `json:"secondary" validate:"omitempty,hexcolor"`
	Success   string `json:"success" validate:"omitempty,hexcolor"`
	Info      string `json:"info" validate:"omitempty,hexcolor"`
	Warning   string `json:"warning" validate:"omitempty,hexcolor"`
	Error     string `json:"error" validate:"omitempty,hexcolor"`
	Neutral   string `json:"neutral" validate:"omitempty,hexcolor"`
}

// TimingSettings is the timing section of the configuration.
type TimingSettings struct {
	SerialPort    string `json:"serialPort" validate:"max=255"`
	BaudRate      int    `json:"baudRate" validate:"oneof=9600 19200 38400 57600 115200"`
	PoolLength    int    `json:"poolLength" validate:"min=0,max=1000000"`
	SimulateEvent int    `json:"simulateEvent" validate:"min=0,max=1000000"`
	ReplayPath    string `json:"replayPath" validate:"max=1024"`
	Recording     bool   `json:"recording"`
}

// WebhookSettings is the webhook section of the configuration.
type WebhookSettings struct {
	AllowedNetworks string `json:"allowedNetworks" validate:"max=1024,networks"`
}

// ConfigSaveRequest is the configuration to save. Its sections are the same as in ConfigResponse.
type ConfigSaveRequest struct {
	App     AppSettings     `json:"app"`
	Log     LogSettings     `json:"log"`
	Server  ServerSettings  `json:"server"`
	Storage StorageSettings `json:"storage"`
	Theme   ThemeSettings   `json:"theme"`
	Timing  TimingSettings  `json:"timing"`
	Webhook WebhookSettings `json:"webhook"`
	// ClearSecrets lists the JSON paths of secrets to remove. A secret sent empty otherwise keeps its value.
	ClearSecrets []string `json:"clearSecrets" validate:"max=50,dive,max=255"`
}

// configSaveHandler validates and writes the application configuration to the config file that was loaded at startup.
// It never applies to the running process. Every service reads its configuration once at startup,
// so a restart is required to apply it.
//
//	@Summary		Save config
//	@Description	Validates the submitted configuration and writes it to the config file. Does not apply to the running process, a restart is required (Admin only).
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ConfigSaveRequest	true	"Application configuration"
//	@Success		200		{object}	ConfigResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/config [put]
func (a *API) configSaveHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[ConfigSaveRequest](w, r)
	if !ok {
		return
	}

	if err := a.configService.Save(req.toConfig(), req.ClearSecrets); err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeConfig(w, r)
}

// writeConfig responds with the current configuration without its secrets.
func (a *API) writeConfig(w http.ResponseWriter, r *http.Request) {
	view, err := a.configService.View()
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
	a.writeJSON(w, r, http.StatusOK, toConfigResponse(view), nil)
}

// toConfigResponse maps the configuration view to the response.
func toConfigResponse(view service.ConfigView) ConfigResponse {
	sources := make(map[string]ConfigSource, len(view.Sources))
	for path, source := range view.Sources {
		sources[path] = ConfigSource{
			Kind: string(source.Kind),
			Name: source.Name,
		}
	}

	return ConfigResponse{
		ConfigSettings: toConfigSettings(view.Config),
		Secrets:        view.Secrets,
		Defaults:       toConfigSettings(view.Defaults),
		Sources:        sources,
		File: ConfigFile{
			Path:   view.File.Path,
			Exists: view.File.Exists,
		},
	}
}

// toConfigSettings maps a configuration to its settings.
func toConfigSettings(cfg config.Config) ConfigSettings {
	return ConfigSettings{
		App: AppSettings{
			Name:     cfg.App.Name,
			Env:      string(cfg.App.Env),
			Language: string(cfg.App.Language),
			Secret:   cfg.App.Secret,
		},
		Log: LogSettings{
			Level:      cfg.Log.RawLevel,
			Channel:    string(cfg.Log.Channel),
			Path:       cfg.Log.Path,
			Format:     string(cfg.Log.Format),
			MaxSizeMB:  cfg.Log.MaxSizeMB,
			MaxAgeDays: cfg.Log.MaxAgeDays,
			Compress:   cfg.Log.Compress,
		},
		Server: ServerSettings{
			Host:                 cfg.Server.Host,
			Port:                 cfg.Server.Port,
			TrustedProxies:       cfg.Server.RawTrustedProxies,
			CORSAllowedOrigins:   cfg.Server.CORSAllowedOrigins,
			CORSAllowCredentials: cfg.Server.CORSAllowCredentials,
			SSLEnabled:           cfg.Server.SSLEnabled,
			SSLCertFile:          cfg.Server.SSLCertFile,
			SSLKeyFile:           cfg.Server.SSLKeyFile,
		},
		Storage: StorageSettings{
			Driver: string(cfg.Storage.Driver),
			Path:   cfg.Storage.Path,
			S3: S3Settings{
				Endpoint:     cfg.Storage.S3.Endpoint,
				Region:       cfg.Storage.S3.Region,
				Bucket:       cfg.Storage.S3.Bucket,
				AccessKey:    cfg.Storage.S3.AccessKey,
				SecretKey:    cfg.Storage.S3.SecretKey,
				UsePathStyle: cfg.Storage.S3.UsePathStyle,
			},
		},
		Theme: toThemeSettings(cfg.Theme),
		Timing: TimingSettings{
			SerialPort:    cfg.Timing.SerialPort,
			BaudRate:      cfg.Timing.BaudRate,
			PoolLength:    cfg.Timing.PoolLength,
			SimulateEvent: cfg.Timing.SimulateEvent,
			ReplayPath:    cfg.Timing.ReplayPath,
			Recording:     cfg.Timing.Recording,
		},
		Webhook: WebhookSettings{
			AllowedNetworks: cfg.Webhook.RawAllowedNetworks,
		},
	}
}

// toThemeSettings maps the theme of the configuration to its settings.
func toThemeSettings(theme config.ThemeConfig) ThemeSettings {
	return ThemeSettings{
		Primary:   theme.Primary,
		Secondary: theme.Secondary,
		Success:   theme.Success,
		Info:      theme.Info,
		Warning:   theme.Warning,
		Error:     theme.Error,
		Neutral:   theme.Neutral,
	}
}

// toConfig maps the request onto a configuration.
func (r ConfigSaveRequest) toConfig() config.Config {
	return config.Config{
		App: config.AppConfig{
			Name:     r.App.Name,
			Env:      config.Environment(r.App.Env),
			Language: config.Language(r.App.Language),
			Secret:   r.App.Secret,
		},
		Log: config.LogConfig{
			RawLevel:   r.Log.Level,
			Channel:    config.LogChannel(r.Log.Channel),
			Path:       r.Log.Path,
			Format:     config.LogFormat(r.Log.Format),
			MaxSizeMB:  r.Log.MaxSizeMB,
			MaxAgeDays: r.Log.MaxAgeDays,
			Compress:   r.Log.Compress,
		},
		Server: config.ServerConfig{
			Host:                 r.Server.Host,
			Port:                 r.Server.Port,
			RawTrustedProxies:    r.Server.TrustedProxies,
			CORSAllowedOrigins:   r.Server.CORSAllowedOrigins,
			CORSAllowCredentials: r.Server.CORSAllowCredentials,
			SSLEnabled:           r.Server.SSLEnabled,
			SSLCertFile:          r.Server.SSLCertFile,
			SSLKeyFile:           r.Server.SSLKeyFile,
		},
		Storage: r.Storage.toConfig(),
		Theme: config.ThemeConfig{
			Primary:   r.Theme.Primary,
			Secondary: r.Theme.Secondary,
			Success:   r.Theme.Success,
			Info:      r.Theme.Info,
			Warning:   r.Theme.Warning,
			Error:     r.Theme.Error,
			Neutral:   r.Theme.Neutral,
		},
		Timing: config.TimingConfig{
			SerialPort:    r.Timing.SerialPort,
			BaudRate:      r.Timing.BaudRate,
			PoolLength:    r.Timing.PoolLength,
			SimulateEvent: r.Timing.SimulateEvent,
			ReplayPath:    r.Timing.ReplayPath,
			Recording:     r.Timing.Recording,
		},
		Webhook: config.WebhookConfig{
			RawAllowedNetworks: r.Webhook.AllowedNetworks,
		},
	}
}

// configStorageTestHandler builds the submitted storage configuration and verifies it is reachable,
// without saving it. A saved configuration only takes effect after a restart.
//
//	@Summary		Test storage config
//	@Description	Builds the submitted storage configuration and verifies it is reachable, without saving it (Admin only).
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		StorageSettings	true	"Storage configuration to test"
//	@Success		200		{object}	ConfigTestResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Security		SessionAuth
//	@Router			/config/storage/test [post]
func (a *API) configStorageTestHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[StorageSettings](w, r)
	if !ok {
		return
	}

	a.writeTestResult(w, r, a.configService.TestStorage(r.Context(), req.toConfig()))
}

// ConfigTestResponse reports whether a driver connection check succeeded.
// A failure is reported with OK false rather than an HTTP error status,
// since an unreachable destination is an expected, actionable outcome of the check.
type ConfigTestResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// toConfig maps the storage settings to the configuration.
func (r StorageSettings) toConfig() config.StorageConfig {
	return config.StorageConfig{
		Driver: media.Driver(r.Driver),
		Path:   r.Path,
		S3: config.S3Config{
			Endpoint:     r.S3.Endpoint,
			Region:       r.S3.Region,
			Bucket:       r.S3.Bucket,
			AccessKey:    r.S3.AccessKey,
			SecretKey:    r.S3.SecretKey,
			UsePathStyle: r.S3.UsePathStyle,
		},
	}
}

// writeTestResult reports the outcome of a connection test. A failed test is a result to show, not a failed request.
func (a *API) writeTestResult(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		a.writeJSON(w, r, http.StatusOK, ConfigTestResponse{
			Message: err.Error(),
		}, nil)
		return
	}
	a.writeJSON(w, r, http.StatusOK, ConfigTestResponse{
		OK: true,
	}, nil)
}
