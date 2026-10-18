package config

import (
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
)

// S3Config holds the S3 storage settings.
type S3Config struct {
	Endpoint     string `mapstructure:"endpoint" json:"endpoint" usage:"S3 endpoint URL of the s3 driver"`
	Region       string `mapstructure:"region" json:"region" usage:"S3 region of the s3 driver"`
	Bucket       string `mapstructure:"bucket" json:"bucket" usage:"S3 bucket of the s3 driver"`
	AccessKey    string `mapstructure:"access_key" json:"accessKey" usage:"S3 access key of the s3 driver"`
	SecretKey    string `mapstructure:"secret_key" json:"secretKey" secret:"true" usage:"S3 secret key of the s3 driver"`
	UsePathStyle bool   `mapstructure:"use_path_style" json:"usePathStyle" default:"true" usage:"Use path-style bucket addressing"`
}

// StorageConfig holds the file storage settings.
type StorageConfig struct {
	Driver media.Driver `mapstructure:"driver" json:"driver" default:"local" values:"local,s3" usage:"Storage driver"`
	Path   string       `mapstructure:"path" json:"path" usage:"Storage root directory of the local driver, empty for the data directory"`
	S3     S3Config     `mapstructure:"s3" json:"s3"`
}

// validateStorage normalizes and checks the file storage settings.
func validateStorage(cfg *StorageConfig) error {
	var p structconf.Problems

	switch cfg.Driver {
	case media.DriverLocal:
		cfg.Path = cleanPath(cfg.Path)
		if len(cfg.Path) > maxPath {
			p.Addf("storage.path must be at most %d characters", maxPath)
		}
	case media.DriverS3:
		if cfg.S3.Endpoint == "" || cfg.S3.Bucket == "" || cfg.S3.AccessKey == "" || cfg.S3.SecretKey == "" {
			p.Addf("storage.driver s3 requires STORAGE_S3_ENDPOINT, STORAGE_S3_BUCKET, STORAGE_S3_ACCESS_KEY and STORAGE_S3_SECRET_KEY")
		}
	}

	return p.Err()
}
