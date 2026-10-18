package domain

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	t.Parallel()

	// The subtests pin the format the overlays read, which must not change
	t.Run("encodes the live heat of an update", func(t *testing.T) {
		t.Parallel()

		raw, err := json.Marshal(NewMessage(MessageUpdate, &LiveHeat{
			Event: 7,
		}))

		require.NoError(t, err)
		fields := decodeMessage(t, raw)
		assert.JSONEq(t, `"update"`, string(fields["type"]))
		var data map[string]jsontext.Value
		require.NoError(t, json.Unmarshal(fields["data"], &data))
		assert.JSONEq(t, `7`, string(data["event"]))
		var serverTime time.Time
		require.NoError(t, json.Unmarshal(fields["serverTime"], &serverTime))
		assert.WithinDuration(t, time.Now(), serverTime, time.Minute)
	})

	t.Run("encodes a meet message without data", func(t *testing.T) {
		t.Parallel()

		raw, err := json.Marshal(NewMessage(MessageMeet, nil))

		require.NoError(t, err)
		fields := decodeMessage(t, raw)
		assert.Equal(t, []string{"serverTime", "type"}, slices.Sorted(maps.Keys(fields)))
		assert.JSONEq(t, `"meet"`, string(fields["type"]))
	})

	t.Run("encodes a missing live heat as null", func(t *testing.T) {
		t.Parallel()

		var heat *LiveHeat

		raw, err := json.Marshal(NewMessage(MessageInit, heat))

		require.NoError(t, err)
		fields := decodeMessage(t, raw)
		assert.JSONEq(t, `"init"`, string(fields["type"]))
		assert.JSONEq(t, `null`, string(fields["data"]))
	})
}

// decodeMessage decodes an encoded message into its top-level fields.
func decodeMessage(t *testing.T, raw []byte) map[string]jsontext.Value {
	t.Helper()

	var fields map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(raw, &fields))
	return fields
}
