package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestValidateApp(t *testing.T) {
	t.Parallel()

	t.Run("trims the name", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(12)
		cfg := testConfig(t)
		cfg.App.Name = "  " + name + "  "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, name, cfg.App.Name)
	})

	t.Run("fills an empty name", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.App.Name = "  "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, AppTitle, cfg.App.Name)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "long name",
				change:  func(c *Config) { c.App.Name = strings.Repeat("a", 101) },
				wantErr: "app.name",
			},
			{
				name:    "unknown language",
				change:  func(c *Config) { c.App.Language = "fr" },
				wantErr: "app.language",
			},
		})
	})

	t.Run("keeps an unknown environment lenient", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.App.Env = "dev"

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, EnvDevelopment, cfg.App.Env)
	})
}

func TestNormalizeEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
		want Environment
	}{
		{
			name: "selects development for dev",
			env:  "dev",
			want: EnvDevelopment,
		},
		{
			name: "selects development for a padded local in capitals",
			env:  " Local ",
			want: EnvDevelopment,
		},
		{
			name: "selects development",
			env:  "development",
			want: EnvDevelopment,
		},
		{
			name: "selects staging",
			env:  "staging",
			want: EnvStaging,
		},
		{
			name: "selects production for prod",
			env:  "prod",
			want: EnvProduction,
		},
		{
			name: "selects production for an empty environment",
			env:  "",
			want: EnvProduction,
		},
		{
			name: "selects production for an unknown environment",
			env:  "unknown",
			want: EnvProduction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := normalizeEnv(tt.env)

			assert.Equal(t, tt.want, env)
		})
	}
}
