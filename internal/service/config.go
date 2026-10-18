package service

import (
	"context"
	"errors"
	"maps"
	"os"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
)

// ConfigService holds the running admin-editable configuration and persists changes to the config file
// that was loaded at startup, or to a newly chosen one if none was loaded.
//
// Saving never applies to the running process: every other service reads its configuration once at startup,
// so a change only takes effect after a restart.
type ConfigService struct {
	mu       sync.RWMutex
	cfg      config.Config
	filePath string
	sources  map[string]structconf.Source
}

// InvalidConfigError reports a configuration that config.Validate rejects as a whole,
// such as a setting whose format only the config package checks.
type InvalidConfigError struct {
	Err error
}

// Error returns the message of the rejected settings.
func (e *InvalidConfigError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *InvalidConfigError) Unwrap() error {
	return e.Err
}

// ConfigOptions holds the settings of the ConfigService.
type ConfigOptions struct {
	// FilePath is the config file loaded at startup, empty if none was found.
	FilePath string
	// Sources reports by JSON path where each setting was loaded from at startup.
	Sources map[string]structconf.Source
}

// NewConfigService creates a ConfigService that starts with cfg as the current configuration.
func NewConfigService(cfg config.Config, options ConfigOptions) *ConfigService {
	return &ConfigService{
		cfg:      cfg,
		filePath: options.FilePath,
		sources:  maps.Clone(options.Sources),
	}
}

// Get returns the current effective configuration.
func (s *ConfigService) Get() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

// Save validates and normalizes cfg, writes it to the config file, and makes it the configuration Get reports.
// A secret submitted empty keeps its current value unless its JSON path is listed in clearSecrets,
// since the admin interface never receives the secrets.
func (s *ConfigService) Save(cfg config.Config, clearSecrets []string) error {
	// The lock covers the whole save, so two saves at once cannot keep the secrets of a configuration
	// the other one replaces in the meantime.
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.cfg
	structconf.KeepSecrets(&cfg, &current, clearSecrets)

	if err := requireSecrets(cfg); err != nil {
		return err
	}

	if err := config.Validate(&cfg); err != nil {
		return &InvalidConfigError{
			Err: err,
		}
	}

	path, err := structconf.Write(config.AppName, s.filePath, &cfg)
	if err != nil {
		return err
	}

	s.cfg = cfg
	s.filePath = path
	for key, source := range s.sources {
		if source.Kind == structconf.SourceDefault || source.Kind == structconf.SourceFile {
			s.sources[key] = structconf.Source{
				Kind: structconf.SourceFile,
				Name: path,
			}
		}
	}

	return nil
}

// requireSecrets reports the secrets the selected drivers need but neither the submitted
// nor the current configuration holds, since the admin interface sends them empty.
func requireSecrets(cfg config.Config) error {
	if cfg.Storage.Driver == media.DriverS3 && cfg.Storage.S3.SecretKey == "" {
		return apperr.NewValidationError("storage.s3.secretKey", "required_if")
	}
	return nil
}

// ConfigView is the configuration as the admin interface shows it, with what it needs to explain each setting.
type ConfigView struct {
	// Config is the current configuration with every secret cleared, so secrets never leave the backend.
	Config config.Config
	// Secrets reports by JSON path whether each secret is set.
	Secrets map[string]bool
	// Defaults is the configuration of the struct-tag defaults, normalized like a loaded configuration,
	// so it compares with Config setting by setting. Secrets have no default and are empty.
	Defaults config.Config
	// Sources reports by JSON path where each setting comes from, see Save.
	Sources map[string]structconf.Source
	// File is the config file a save writes to.
	File ConfigFile
}

// ConfigFile is the config file the configuration is loaded from and saved to.
type ConfigFile struct {
	// Path is empty if no config file was found at startup. A save then creates one, see structconf.Write.
	Path string
	// Exists reports whether the file exists, which is false for a path set before the file is created.
	Exists bool
}

// View returns the current configuration for the admin interface with the defaults,
// the source of each setting, and the config file.
func (s *ConfigService) View() (ConfigView, error) {
	defaults, err := config.Defaults()
	if err != nil {
		return ConfigView{}, err
	}
	// Validating normalizes the defaults like a loaded configuration. A setting without a default
	// that a selected driver requires fails the check, which does not matter for a comparison.
	_ = config.Validate(&defaults)
	structconf.HideSecrets(&defaults)

	s.mu.RLock()
	cfg := s.cfg
	sources := maps.Clone(s.sources)
	path := s.filePath
	s.mu.RUnlock()

	secrets := structconf.HideSecrets(&cfg)

	return ConfigView{
		Config:   cfg,
		Secrets:  secrets,
		Defaults: defaults,
		Sources:  sources,
		File:     configFileOf(path),
	}, nil
}

// configFileOf describes the config file at path, which may be empty or not exist yet.
func configFileOf(path string) ConfigFile {
	if path == "" {
		return ConfigFile{}
	}

	_, err := os.Stat(path)
	return ConfigFile{
		Path:   path,
		Exists: err == nil,
	}
}

// TestStorage checks that the storage of cfg is reachable without saving cfg or changing the storage.
// An empty secret key tests the saved key, and an empty local path tests the data directory the application falls back to.
func (s *ConfigService) TestStorage(ctx context.Context, cfg config.StorageConfig) error {
	if cfg.S3.SecretKey == "" {
		cfg.S3.SecretKey = s.Get().Storage.S3.SecretKey
	}

	if cfg.Driver != media.DriverLocal {
		return media.Test(ctx, storageTestOptions(cfg, cfg.Path))
	}

	// The application keeps its data in the first candidate it can write to, so one writable candidate passes
	paths := config.DataPaths(cfg.Path, "")
	if len(paths) == 0 {
		return errors.New("no data directory available")
	}
	var errs []error
	for _, path := range paths {
		err := media.Test(ctx, storageTestOptions(cfg, path))
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// storageTestOptions maps the storage settings of cfg with localPath as the local directory to the media options.
func storageTestOptions(cfg config.StorageConfig, localPath string) media.Options {
	return media.Options{
		Driver:         cfg.Driver,
		LocalPath:      localPath,
		S3Endpoint:     cfg.S3.Endpoint,
		S3Region:       cfg.S3.Region,
		S3Bucket:       cfg.S3.Bucket,
		S3AccessKey:    cfg.S3.AccessKey,
		S3SecretKey:    cfg.S3.SecretKey,
		S3UsePathStyle: cfg.S3.UsePathStyle,
	}
}
