package service

import (
	"encoding/json/v2"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/domain"
)

// broadcaster publishes a message to every connected client.
type broadcaster interface {
	Broadcast(message []byte)
}

// announceChange tells the connected clients that a resource changed, so they fetch it again.
// A failed announcement is only logged, since the clients still see the change on their next fetch.
func announceChange(logger zerolog.Logger, broadcaster broadcaster, resource domain.Resource, id string) {
	message, err := json.Marshal(domain.NewMessage(domain.MessageChanged, domain.Changed{
		Resource: resource,
		ID:       id,
	}))
	if err != nil {
		logger.Error().Err(err).Str("resource", string(resource)).Msg("failed to encode changed message")
		return
	}
	broadcaster.Broadcast(message)
}
