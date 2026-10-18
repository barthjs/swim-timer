package structconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// valuesConfig is a sample configuration with restricted values.
type valuesConfig struct {
	Mode  string `mapstructure:"mode" values:"fast,slow" usage:"Mode"`
	Speed int    `mapstructure:"speed" values:"1,2" usage:"Speed"`
	Free  string `mapstructure:"free" usage:"Free text"`
}

func TestCheckValues(t *testing.T) {
	t.Parallel()

	t.Run("accepts the allowed values", func(t *testing.T) {
		t.Parallel()

		err := CheckValues(&valuesConfig{
			Mode:  "fast",
			Speed: 2,
			Free:  "anything",
		})

		assert.NoError(t, err)
	})

	t.Run("rejects values outside the list", func(t *testing.T) {
		t.Parallel()

		err := CheckValues(&valuesConfig{
			Mode:  "medium",
			Speed: 3,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, `mode must be one of fast, slow, got "medium"`)
		assert.ErrorContains(t, err, `speed must be one of 1, 2, got "3"`)
	})
}
