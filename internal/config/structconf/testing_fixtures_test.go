package structconf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// tagConfig exercises every field type and tag combination the package supports.
type tagConfig struct {
	Demo tagDemo `mapstructure:"demo" json:"demo"`
}

// tagDemo is a sample configuration section that covers the supported field types and tags.
type tagDemo struct {
	Name     string        `mapstructure:"name" json:"name" default:"demo" usage:"Name of the demo"`
	Debug    bool          `mapstructure:"debug" json:"debug" default:"true" usage:"Debug mode"`
	Ratio    float64       `mapstructure:"ratio" json:"ratio" default:"0.5" usage:"A ratio"`
	Timeout  time.Duration `mapstructure:"timeout" json:"timeout" default:"1m30s" usage:"A timeout"`
	Tags     []string      `mapstructure:"tags" json:"tags" default:"a,b" usage:"Some tags"`
	Server   tagServer     `mapstructure:"server" json:"server"`
	Derived  int           `mapstructure:"-"`
	Untagged string
}

// tagServer is a nested sample configuration section.
type tagServer struct {
	Port     int    `mapstructure:"port" json:"port" default:"8080" usage:"Port"`
	LogLevel string `mapstructure:"log_level" json:"logLevel" default:"warn" values:"debug, info,warn,error" usage:"Log level"`
	NoValue  string `mapstructure:"no_value" usage:"Without a default"`
}

// testAppName names the config directories the tests search, which do not exist on the test machine.
const testAppName = "structconf-test"

// loadTagConfig loads a tagConfig through all sources.
func loadTagConfig(t *testing.T, flags *pflag.FlagSet) (tagConfig, error) {
	t.Helper()

	var cfg tagConfig
	err := Load(testAppName, flags, &cfg)
	return cfg, err
}

// writeFile writes content to a file in a fresh temporary directory and returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
