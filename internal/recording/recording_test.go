package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestNewMeetFile(t *testing.T) {
	t.Parallel()

	t.Run("keeps the content with its hash", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		name := faker.LetterN(8)
		content := []byte(faker.Sentence(5))

		got := NewMeetFile(name, content)

		assert.Equal(t, name, got.Name)
		assert.Equal(t, content, got.Content)
		assert.Equal(t, Hash(content), got.SHA256)
	})
}

func TestHash(t *testing.T) {
	t.Parallel()

	t.Run("returns the hex encoded SHA-256 hash", func(t *testing.T) {
		t.Parallel()

		content := []byte(apitest.Faker(t).Sentence(5))
		sum := sha256.Sum256(content)

		got := Hash(content)

		assert.Equal(t, hex.EncodeToString(sum[:]), got)
	})
}

func TestHexBytes_MarshalText(t *testing.T) {
	t.Parallel()

	t.Run("writes space-separated upper-case hex bytes", func(t *testing.T) {
		t.Parallel()

		got, err := HexBytes{0x01, 0x02, 0x1E, 0xFF}.MarshalText()

		require.NoError(t, err)
		assert.Equal(t, "01 02 1E FF", string(got))
	})

	t.Run("writes a JSON string", func(t *testing.T) {
		t.Parallel()

		got, err := json.Marshal(HexBytes{0x04, 0xA0})

		require.NoError(t, err)
		assert.JSONEq(t, `"04 A0"`, string(got))
	})
}

func TestHexBytes_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want HexBytes
	}{
		{
			name: "reads space-separated hex bytes",
			text: "01 02 1E FF",
			want: HexBytes{0x01, 0x02, 0x1E, 0xFF},
		},
		{
			name: "reads hex bytes without spaces",
			text: "01021eff",
			want: HexBytes{0x01, 0x02, 0x1E, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got HexBytes
			err := got.UnmarshalText([]byte(tt.text))

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects text that is not hex", func(t *testing.T) {
		t.Parallel()

		var got HexBytes
		err := got.UnmarshalText([]byte("01 0G"))

		assert.Error(t, err)
	})
}
