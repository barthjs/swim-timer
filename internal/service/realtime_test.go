package service

import (
	"encoding/json/v2"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/domain"
)

func TestAnnounceChange(t *testing.T) {
	t.Parallel()

	t.Run("broadcasts the changed resource", func(t *testing.T) {
		t.Parallel()
		broadcaster := &fakeBroadcaster{}

		announceChange(zerolog.Nop(), broadcaster, domain.ResourcePreset, "main")

		require.Equal(t, 1, broadcaster.count())
		assertChanged(t, broadcaster.messages[0], domain.ResourcePreset, "main")
	})
}

// assertChanged checks that a message announces the change of the resource and the item.
func assertChanged(t *testing.T, raw []byte, resource domain.Resource, id string) {
	t.Helper()

	var message struct {
		Type domain.MessageType `json:"type"`
		Data domain.Changed     `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &message))
	assert.Equal(t, domain.MessageChanged, message.Type)
	assert.Equal(t, domain.Changed{
		Resource: resource,
		ID:       id,
	}, message.Data)
}
