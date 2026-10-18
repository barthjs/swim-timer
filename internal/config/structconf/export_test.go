package structconf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToMap(t *testing.T) {
	t.Parallel()

	t.Run("returns the values by key without derived fields", func(t *testing.T) {
		t.Parallel()

		type server struct {
			Port    int           `mapstructure:"port"`
			Timeout time.Duration `mapstructure:"timeout"`
		}
		type demo struct {
			Name    string `mapstructure:"name"`
			Server  server `mapstructure:"server"`
			Derived int    `mapstructure:"-"`
		}

		got := ToMap(&demo{
			Name: "demo",
			Server: server{
				Port:    8080,
				Timeout: 30 * time.Second,
			},
			Derived: 1,
		})

		assert.Equal(t, map[string]any{
			"name": "demo",
			"server": map[string]any{
				"port":    8080,
				"timeout": "30s",
			},
		}, got)
	})
}
