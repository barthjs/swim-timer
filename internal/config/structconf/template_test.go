package structconf

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// templateConfig is a sample configuration for the template tests.
type templateConfig struct {
	Demo   tagDemo        `mapstructure:"demo"`
	Access templateAccess `mapstructure:"access"`
}

// templateAccess is a sample configuration section with a secret field.
type templateAccess struct {
	Token string `mapstructure:"token" default:"not-written" secret:"true" usage:"Access token"`
	Mode  string `mapstructure:"mode" default:"*" values:"*,read" usage:"Access mode"`
}

func TestTemplate(t *testing.T) {
	t.Parallel()

	t.Run("writes the commented defaults", func(t *testing.T) {
		t.Parallel()

		var sb strings.Builder

		require.NoError(t, Template(&sb, &templateConfig{}))

		out := sb.String()
		assert.True(t, strings.HasPrefix(out, "demo:\n  # Name of the demo\n  name: demo\n"), out)
		assert.Contains(t, out, "  server:\n    # Port\n    port: 8080\n", "nested sections are indented")
		assert.Contains(t, out, "    # Log level (debug, info, warn, error)\n", "the comment lists the allowed values")
		assert.Contains(t, out, "  timeout: 1m30s\n", "durations are written as text")
		assert.Contains(t, out, "  tags: [a, b]\n")
		assert.Contains(t, out, "\naccess:\n", "a new top-level section starts after a blank line")
		assert.Contains(t, out, "  token: \"\"\n", "secrets stay empty")
		assert.NotContains(t, out, "not-written")
	})

	t.Run("loads back to the defaults without secrets", func(t *testing.T) {
		t.Parallel()

		var sb strings.Builder

		require.NoError(t, Template(&sb, &templateConfig{}))

		path := writeFile(t, "config.yaml", sb.String())
		var loaded templateConfig
		require.NoError(t, LoadFile(path, &loaded))
		var defaults templateConfig
		require.NoError(t, Defaults(&defaults))
		defaults.Access.Token = ""
		assert.Equal(t, defaults, loaded)
	})
}
