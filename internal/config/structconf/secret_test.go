package structconf

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
)

// secretConfig is a configuration with secrets in nested sections.
type secretConfig struct {
	App     secretApp     `json:"app"`
	Storage secretStorage `json:"storage"`
	Ignored string        `json:"-" secret:"true"`
}

// secretApp is a section with a string secret and a tagged field that is no string.
type secretApp struct {
	Name     string `json:"name"`
	Password string `json:"password" secret:"true"`
	Port     int    `json:"port" secret:"true"`
}

// secretStorage is a section with an optional secret.
type secretStorage struct {
	Token  string `json:"token,omitempty" secret:"true"`
	Bucket string `json:"bucket"`
}

func TestRedacted(t *testing.T) {
	t.Parallel()

	t.Run("replaces the secrets in a copy", func(t *testing.T) {
		t.Parallel()

		cfg := newSecretConfig(apitest.Faker(t))
		password := cfg.App.Password

		redacted := Redacted(cfg)

		assert.Equal(t, redactedValue, redacted.App.Password)
		assert.Empty(t, redacted.Storage.Token, "an empty secret stays empty")
		assert.Equal(t, cfg.App.Name, redacted.App.Name)
		assert.Equal(t, cfg.App.Port, redacted.App.Port, "only strings are secrets")
		assert.Equal(t, cfg.Ignored, redacted.Ignored, "fields without a JSON name are skipped")
		assert.Equal(t, password, cfg.App.Password, "the original is unchanged")
	})
}

func TestHideSecrets(t *testing.T) {
	t.Parallel()

	t.Run("clears the secrets and reports which were set", func(t *testing.T) {
		t.Parallel()

		cfg := newSecretConfig(apitest.Faker(t))
		bucket := cfg.Storage.Bucket

		set := HideSecrets(&cfg)

		assert.Equal(t, map[string]bool{"app.password": true, "storage.token": false}, set)
		assert.Empty(t, cfg.App.Password)
		assert.Equal(t, bucket, cfg.Storage.Bucket)
	})
}

func TestKeepSecrets(t *testing.T) {
	t.Parallel()

	faker := apitest.Faker(t)
	current := newSecretConfig(faker)
	current.Storage.Token = faker.LetterN(16)
	password := current.App.Password
	token := current.Storage.Token
	changed := faker.LetterN(16)

	tests := []struct {
		name         string
		submitted    secretApp
		clear        []string
		wantPassword string
		wantToken    string
	}{
		{
			name:         "keeps the current value for an empty secret",
			wantPassword: password,
			wantToken:    token,
		},
		{
			name: "replaces the secret with a new value",
			submitted: secretApp{
				Password: changed,
			},
			wantPassword: changed,
			wantToken:    token,
		},
		{
			name:  "removes the listed secrets",
			clear: []string{"app.password", "storage.token"},
		},
		{
			name:         "ignores unknown paths",
			clear:        []string{"app.name"},
			wantPassword: password,
			wantToken:    token,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			submitted := secretConfig{
				App: tt.submitted,
			}

			KeepSecrets(&submitted, &current, tt.clear)

			assert.Equal(t, tt.wantPassword, submitted.App.Password)
			assert.Equal(t, tt.wantToken, submitted.Storage.Token)
		})
	}
}

// newSecretConfig returns a configuration with a password and without a token.
func newSecretConfig(faker *gofakeit.Faker) secretConfig {
	return secretConfig{
		App: secretApp{
			Name:     faker.LetterN(8),
			Password: faker.LetterN(16),
			Port:     faker.IntRange(1, 65535),
		},
		Storage: secretStorage{
			Bucket: faker.LetterN(8),
		},
		Ignored: faker.LetterN(8),
	}
}
