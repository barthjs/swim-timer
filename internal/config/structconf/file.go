package structconf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

// fileExtensions are the config file formats, in the order FindFile prefers them.
var fileExtensions = []string{"yaml", "yml", "json", "toml"}

// FindFile returns the path of the config file Load reads, or an empty path for Write to pick a location.
func FindFile(appName string, flags *pflag.FlagSet) string {
	if explicit := explicitFile(flags); explicit != "" {
		return explicit
	}

	for _, path := range SearchPaths(appName) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// SearchPaths returns the config file paths FindFile tries without an explicit file, in order.
func SearchPaths(appName string) []string {
	var paths []string
	for _, dir := range appdirs.ConfigSearchDirs(appName, "") {
		for _, ext := range fileExtensions {
			paths = append(paths, filepath.Join(dir, fileName+"."+ext))
		}
	}
	return paths
}

// Write replaces the config file at path, or at a new path in a config directory of appName for an empty one,
// with the settings of cfg and returns the path.
func Write(appName, path string, cfg any) (string, error) {
	if path == "" {
		var err error
		path, err = newFilePath(appName)
		if err != nil {
			return "", err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config directory %q: %w", dir, err)
	}

	v := viper.New()
	eachLeafValue(cfg, func(key string, value any) { v.Set(key, value) })

	// A temporary file renamed over the old one keeps a reader from seeing half a file.
	// WriteConfigAs infers the file format (YAML, JSON, TOML) from the target's extension,
	// so the temp file must share path's extension.
	tmp, err := os.CreateTemp(dir, fileName+"-*"+filepath.Ext(path))
	if err != nil {
		return "", fmt.Errorf("creating temporary config file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := v.WriteConfigAs(tmpPath); err != nil {
		return "", fmt.Errorf("writing config: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return "", fmt.Errorf("replacing config file %q: %w", path, err)
	}

	return path, nil
}

// newFilePath picks the location of a new config file: the first writable of the config directories,
// tried in the order a fresh installation prefers them.
func newFilePath(appName string) (string, error) {
	candidates := appdirs.ConfigWriteDirs(appName, "")

	for _, dir := range candidates {
		if appdirs.DirWritable(dir) {
			return filepath.Join(dir, fileName+".yaml"), nil
		}
	}

	return "", fmt.Errorf("no writable location found for a new config file (tried %v)", candidates)
}
