// Package structconf loads a configuration struct from command-line flags, environment variables,
// a config file, and defaults. Every setting is a struct field whose tags declare it once:
// `mapstructure` names the dotted config-file key, `default` gives the fallback value, `usage` documents it,
// `values` lists the allowed values, and `secret:"true"` keeps its value out of logs.
// The flag and the environment variable are derived from the key.
package structconf

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

const (
	// FileFlag is the command-line flag that names the config file.
	FileFlag = "config"
	// FileEnv is the environment variable that names the config file.
	FileEnv = "CONFIG_FILE"
	// fileName is the name of a config file without its extension.
	fileName = "config"
)

// Load fills the struct cfg points to from four sources, in decreasing precedence:
// command-line flags, environment variables, a config file, and the defaults of the struct tags.
// appName names the directory the config file is searched in (see FindFile).
// The flags may be nil for a command that registers none. The flag layer is then skipped.
func Load(appName string, flags *pflag.FlagSet, cfg any) error {
	v := viper.New()

	if err := bindLeaves(v, cfg, flags); err != nil {
		return err
	}

	if err := readFile(v, appName, flags); err != nil {
		return err
	}

	return v.Unmarshal(cfg)
}

// Defaults fills the struct cfg points to from the defaults of its struct tags alone,
// so it holds the same values on every machine.
func Defaults(cfg any) error {
	v := viper.New()

	if err := setDefaults(v, cfg); err != nil {
		return err
	}

	return v.Unmarshal(cfg)
}

// LoadFile fills the struct cfg points to from the defaults and the config file at path alone,
// without the environment or flags, so a file can be checked on its own. A missing file is an error.
func LoadFile(path string, cfg any) error {
	v := viper.New()

	if err := setDefaults(v, cfg); err != nil {
		return err
	}

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config file %q: %w", path, err)
	}

	return v.Unmarshal(cfg)
}

// readFile makes viper read an optional config file.
//
// An explicit path from the flag or the environment takes precedence, but a missing file there is not an error,
// since container images set the variable to a fixed path that may not exist yet on a fresh volume.
// Without an explicit path, the config directories of appName are searched.
// A missing file there is not an error either.
func readFile(v *viper.Viper, appName string, flags *pflag.FlagSet) error {
	if explicit := explicitFile(flags); explicit != "" {
		v.SetConfigFile(explicit)
		if err := v.ReadInConfig(); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("reading config file %q: %w", explicit, err)
		}
		return nil
	}

	v.SetConfigName(fileName)
	for _, dir := range appdirs.ConfigSearchDirs(appName, "") {
		v.AddConfigPath(dir)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return fmt.Errorf("reading config file: %w", err)
		}
	}
	return nil
}

// explicitFile returns the config file named by the flag, else by the environment, or empty if neither names one.
func explicitFile(flags *pflag.FlagSet) string {
	if flags != nil {
		if path, err := flags.GetString(FileFlag); err == nil && path != "" {
			return path
		}
	}
	return os.Getenv(FileEnv)
}
